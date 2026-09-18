package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/auth"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/contracts"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/requestid"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
	maxCreateBody    = 1 << 20
)

type payloadTooLargeError struct{ message string }

func (err payloadTooLargeError) Error() string { return err.message }

var (
	sandboxIDPattern   = regexp.MustCompile(`^sbx_[A-Za-z0-9_-]+$`)
	commandIDPattern   = regexp.MustCompile(`^cmd_[A-Za-z0-9_-]+$`)
	snapshotIDPattern  = regexp.MustCompile(`^snp_[A-Za-z0-9_-]+$`)
	idempotencyPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)
)

type principalKey struct{}

type Server struct {
	service             *sandbox.Service
	auth                sandbox.AuthService
	idempotency         *idempotencyStore
	snapshotIdempotency *snapshotIdempotencyStore
	heartbeatInterval   time.Duration
}

func NewServer(service *sandbox.Service, authentication sandbox.AuthService) *Server {
	return NewServerWithHeartbeat(service, authentication, 15*time.Second)
}

func NewServerWithHeartbeat(service *sandbox.Service, authentication sandbox.AuthService, heartbeatInterval time.Duration) *Server {
	if heartbeatInterval <= 0 {
		heartbeatInterval = 15 * time.Second
	}
	return &Server{
		service:             service,
		auth:                authentication,
		idempotency:         &idempotencyStore{entries: make(map[string]idempotencyEntry)},
		snapshotIdempotency: &snapshotIdempotencyStore{entries: make(map[string]snapshotIdempotencyEntry)},
		heartbeatInterval:   heartbeatInterval,
	}
}

func (server *Server) WithHeartbeat(interval time.Duration) *Server {
	if interval > 0 {
		server.heartbeatInterval = interval
	}
	return server
}

func (server *Server) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler := requestid.Middleware(http.HandlerFunc(server.route))
	handler.ServeHTTP(writer, request)
}

func PrincipalFromContext(ctx context.Context) (sandbox.Principal, bool) {
	principal, ok := ctx.Value(principalKey{}).(sandbox.Principal)
	return principal, ok
}

func (server *Server) route(writer http.ResponseWriter, request *http.Request) {
	segments := strings.Split(strings.Trim(request.URL.Path, "/"), "/")
	if len(segments) < 2 || segments[0] != "v1" || (segments[1] != "sandboxes" && segments[1] != "snapshots") {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Resource was not found.", nil)
		return
	}
	principal, ok := server.authenticate(writer, request)
	if !ok {
		return
	}
	request = request.WithContext(context.WithValue(request.Context(), principalKey{}, principal))

	switch {
	case len(segments) == 2 && request.Method == http.MethodPost:
		server.createSandbox(writer, request, principal)
	case len(segments) == 2 && request.Method == http.MethodGet:
		server.listSandboxes(writer, request, principal)
	case len(segments) == 4 && segments[3] == "commands" && request.Method == http.MethodPost:
		server.executeCommand(writer, request, principal, segments[2])
	case len(segments) == 6 && segments[3] == "commands" && segments[5] == "events" && request.Method == http.MethodGet:
		server.streamCommandEvents(writer, request, principal, segments[2], segments[4])
	case len(segments) == 4 && segments[3] == "files" && request.Method == http.MethodGet:
		server.listFiles(writer, request, principal, segments[2])
	case len(segments) == 4 && segments[3] == "snapshots":
		switch request.Method {
		case http.MethodPost:
			server.createSnapshot(writer, request, principal, segments[2])
		case http.MethodGet:
			server.listSnapshots(writer, request, principal, segments[2])
		default:
			server.writeError(writer, request, http.StatusNotFound, "not_found", "Resource was not found.", nil)
		}
	case len(segments) == 5 && segments[3] == "files" && segments[4] == "content":
		switch request.Method {
		case http.MethodGet:
			server.readFile(writer, request, principal, segments[2])
		case http.MethodPut:
			server.writeFile(writer, request, principal, segments[2])
		case http.MethodDelete:
			server.deleteFile(writer, request, principal, segments[2])
		default:
			server.writeError(writer, request, http.StatusNotFound, "not_found", "Resource was not found.", nil)
		}
	case len(segments) == 3 && segments[1] == "snapshots" && request.Method == http.MethodGet:
		server.getSnapshot(writer, request, principal, segments[2])
	case len(segments) == 4 && segments[3] == "restore" && request.Method == http.MethodPost:
		server.restoreSnapshot(writer, request, principal, segments[2])
	case len(segments) == 3 && request.Method == http.MethodGet:
		server.getSandbox(writer, request, principal, segments[2])
	case len(segments) == 3 && request.Method == http.MethodDelete:
		server.destroySandbox(writer, request, principal, segments[2])
	default:
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Resource was not found.", nil)
	}
}

func (server *Server) authenticate(writer http.ResponseWriter, request *http.Request) (sandbox.Principal, bool) {
	parts := strings.Fields(request.Header.Get("Authorization"))
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || parts[1] == "" || server.auth == nil {
		server.writeError(writer, request, http.StatusUnauthorized, "authentication_required", "A valid API key is required.", nil)
		return sandbox.Principal{}, false
	}
	principal, err := server.auth.Authenticate(request.Context(), parts[1])
	if err == nil {
		return principal, true
	}
	if errors.Is(err, auth.ErrInvalidAPIKey) {
		server.writeError(writer, request, http.StatusUnauthorized, "authentication_required", "A valid API key is required.", nil)
		return sandbox.Principal{}, false
	}
	server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
	return sandbox.Principal{}, false
}

func (server *Server) createSandbox(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal) {
	var input contracts.SandboxCreateRequest
	if err := decodeJSON(writer, request, &input, maxCreateBody); err != nil {
		server.writeValidationError(writer, request, err)
		return
	}
	config, err := parseSandboxConfig(input.Config)
	if err != nil {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	idempotencyKey := request.Header.Get("Idempotency-Key")
	if !idempotencyPattern.MatchString(idempotencyKey) {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "Idempotency-Key must contain 1 to 128 safe characters.", nil)
		return
	}
	fingerprint, _ := json.Marshal(input)
	sandboxValue, err := server.idempotency.do(principal.OwnerID, idempotencyKey, fingerprint, func() (sandbox.Sandbox, error) {
		if server.service == nil {
			return sandbox.Sandbox{}, errors.New("sandbox service is unavailable")
		}
		return server.service.Create(request.Context(), principal.OwnerID, config)
	})
	if errors.Is(err, errIdempotencyConflict) {
		server.writeError(writer, request, http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used with a different request.", nil)
		return
	}
	if err != nil {
		server.writeServiceError(writer, request, err)
		return
	}
	location := sandboxPath(sandboxValue.ID)
	writer.Header().Set("Location", location)
	server.writeJSON(writer, http.StatusAccepted, toContractSandbox(sandboxValue))
}

func (server *Server) listSandboxes(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal) {
	limit, err := queryLimit(request.URL.Query().Get("limit"))
	if err != nil {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	cursor := request.URL.Query().Get("cursor")
	if len(cursor) > 512 {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "cursor must be at most 512 characters.", nil)
		return
	}
	state := request.URL.Query().Get("state")
	if state != "" && !validState(sandbox.State(state)) {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "state is invalid.", nil)
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	page, err := server.service.List(request.Context(), principal.OwnerID, cursor, limit)
	if err != nil {
		server.writeServiceError(writer, request, err)
		return
	}
	items := make([]contracts.Sandbox, 0, len(page.Items))
	for _, value := range page.Items {
		if state == "" || value.State == sandbox.State(state) {
			items = append(items, toContractSandbox(value))
		}
	}
	response := contracts.SandboxPage{Items: items, Page: contracts.PageInfo{HasMore: page.NextCursor != ""}}
	if page.NextCursor != "" {
		response.Page.NextCursor = &page.NextCursor
	}
	server.writeJSON(writer, http.StatusOK, response)
}

func (server *Server) getSandbox(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, ok := parseSandboxID(id)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	value, err := server.service.Get(request.Context(), principal.OwnerID, sandboxID)
	if err != nil {
		server.writeServiceError(writer, request, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, toContractSandbox(value))
}

func (server *Server) destroySandbox(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, ok := parseSandboxID(id)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	value, err := server.service.Destroy(request.Context(), principal.OwnerID, sandboxID)
	if err != nil {
		server.writeServiceError(writer, request, err)
		return
	}
	writer.Header().Set("Location", sandboxPath(value.ID))
	server.writeJSON(writer, http.StatusAccepted, toContractSandbox(value))
}

func (server *Server) createSnapshot(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, ok := parseSandboxID(id)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	var input contracts.SnapshotCreateRequest
	if err := decodeOptionalJSON(writer, request, &input, maxCreateBody); err != nil {
		server.writeValidationError(writer, request, err)
		return
	}
	idempotencyKey := request.Header.Get("Idempotency-Key")
	if !idempotencyPattern.MatchString(idempotencyKey) {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "Idempotency-Key must contain 1 to 128 safe characters.", nil)
		return
	}
	fingerprint, _ := json.Marshal(input)
	snapshot, err := server.snapshotIdempotency.do(principal.OwnerID, idempotencyKey, fingerprint, func() (sandbox.SnapshotMetadata, error) {
		if server.service == nil {
			return sandbox.SnapshotMetadata{}, errors.New("sandbox service is unavailable")
		}
		return server.service.CreateSnapshot(request.Context(), principal.OwnerID, sandboxID, input.ExpiresAt)
	})
	if errors.Is(err, errIdempotencyConflict) {
		server.writeError(writer, request, http.StatusConflict, "idempotency_conflict", "Idempotency-Key was already used with a different request.", nil)
		return
	}
	if err != nil {
		server.writeServiceError(writer, request, err)
		return
	}
	writer.Header().Set("Location", snapshotPath(snapshot.ID))
	server.writeJSON(writer, http.StatusAccepted, toContractSnapshot(snapshot))
}

func (server *Server) listSnapshots(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, ok := parseSandboxID(id)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	limit, err := queryLimit(request.URL.Query().Get("limit"))
	if err != nil {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", err.Error(), nil)
		return
	}
	cursor := request.URL.Query().Get("cursor")
	if len(cursor) > 512 {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "cursor must be at most 512 characters.", nil)
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	page, err := server.service.ListSnapshots(request.Context(), principal.OwnerID, sandboxID, cursor, limit)
	if err != nil {
		server.writeServiceError(writer, request, err)
		return
	}
	items := make([]contracts.SnapshotMetadata, len(page.Items))
	for index, snapshot := range page.Items {
		items[index] = toContractSnapshot(snapshot)
	}
	response := contracts.SnapshotPage{Items: items, Page: contracts.PageInfo{HasMore: page.NextCursor != ""}}
	if page.NextCursor != "" {
		response.Page.NextCursor = &page.NextCursor
	}
	server.writeJSON(writer, http.StatusOK, response)
}

func (server *Server) getSnapshot(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	snapshotID, ok := parseSnapshotID(id)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Snapshot was not found.", nil)
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	snapshot, err := server.service.GetSnapshot(request.Context(), principal.OwnerID, snapshotID)
	if err != nil {
		server.writeServiceError(writer, request, err)
		return
	}
	server.writeJSON(writer, http.StatusOK, toContractSnapshot(snapshot))
}

func (server *Server) restoreSnapshot(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, ok := parseSandboxID(id)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	var input contracts.RestoreRequest
	if err := decodeJSON(writer, request, &input, maxCreateBody); err != nil {
		server.writeValidationError(writer, request, err)
		return
	}
	snapshotID, ok := parseSnapshotID(input.SnapshotID)
	if !ok {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "snapshotId is invalid.", nil)
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	if err := server.service.Restore(request.Context(), principal.OwnerID, sandboxID, snapshotID); err != nil {
		server.writeServiceError(writer, request, err)
		return
	}
	value, err := server.service.Get(request.Context(), principal.OwnerID, sandboxID)
	if err != nil {
		server.writeServiceError(writer, request, err)
		return
	}
	writer.Header().Set("Location", sandboxPath(value.ID))
	server.writeJSON(writer, http.StatusAccepted, toContractSandbox(value))
}

func decodeJSON(writer http.ResponseWriter, request *http.Request, destination any, maxBytes int64) error {
	request.Body = http.MaxBytesReader(writer, request.Body, maxBytes)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return payloadTooLargeError{message: "Request body exceeds the configured limit."}
		}
		return fmt.Errorf("request body is malformed: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request body must contain one JSON value")
	}
	return nil
}

func decodeOptionalJSON(writer http.ResponseWriter, request *http.Request, destination any, maxBytes int64) error {
	if request.Body == nil || request.Body == http.NoBody || request.ContentLength == 0 {
		return nil
	}
	return decodeJSON(writer, request, destination, maxBytes)
}

func (server *Server) writeValidationError(writer http.ResponseWriter, request *http.Request, err error) {
	var tooLarge payloadTooLargeError
	if errors.As(err, &tooLarge) {
		server.writeError(writer, request, http.StatusRequestEntityTooLarge, "payload_too_large", tooLarge.Error(), nil)
		return
	}
	server.writeError(writer, request, http.StatusBadRequest, "invalid_request", err.Error(), nil)
}

func parseSandboxConfig(input contracts.SandboxConfig) (sandbox.SandboxConfig, error) {
	if input.Image == "" || len(input.Image) > 256 {
		return sandbox.SandboxConfig{}, errors.New("config.image must contain 1 to 256 characters")
	}
	if input.CpuMillis < 128 || input.CpuMillis > 8192 {
		return sandbox.SandboxConfig{}, errors.New("config.cpuMillis must be between 128 and 8192")
	}
	if input.MemoryMiB < 256 || input.MemoryMiB > 32768 {
		return sandbox.SandboxConfig{}, errors.New("config.memoryMiB must be between 256 and 32768")
	}
	if input.StorageGiB < 1 || input.StorageGiB > 200 {
		return sandbox.SandboxConfig{}, errors.New("config.storageGiB must be between 1 and 200")
	}
	if input.MaxLifetimeSeconds < 60 || input.MaxLifetimeSeconds > 86400 {
		return sandbox.SandboxConfig{}, errors.New("config.maxLifetimeSeconds must be between 60 and 86400")
	}
	if input.DefaultCommandTimeoutSeconds < 1 || input.DefaultCommandTimeoutSeconds > 900 {
		return sandbox.SandboxConfig{}, errors.New("config.defaultCommandTimeoutSeconds must be between 1 and 900")
	}
	if len(input.Environment) > 32 {
		return sandbox.SandboxConfig{}, payloadTooLargeError{message: "config.environment exceeds the 32 entry limit."}
	}
	for key, value := range input.Environment {
		if key == "" {
			return sandbox.SandboxConfig{}, errors.New("config.environment keys and values are invalid")
		}
		if len(value) > 4096 {
			return sandbox.SandboxConfig{}, payloadTooLargeError{message: "config.environment value exceeds the 4096 byte limit."}
		}
	}
	repository, err := parseRepository(input.Repository)
	if err != nil {
		return sandbox.SandboxConfig{}, err
	}
	var snapshotID *sandbox.SnapshotID
	if input.SnapshotID != nil {
		value := sandbox.SnapshotID(*input.SnapshotID)
		if !strings.HasPrefix(string(value), "snp_") {
			return sandbox.SandboxConfig{}, errors.New("config.snapshotId is invalid")
		}
		snapshotID = &value
	}
	return sandbox.SandboxConfig{
		Image:                 input.Image,
		CPUMillis:             int(input.CpuMillis),
		MemoryMiB:             int(input.MemoryMiB),
		StorageGiB:            int(input.StorageGiB),
		MaxLifetime:           time.Duration(input.MaxLifetimeSeconds) * time.Second,
		DefaultCommandTimeout: time.Duration(input.DefaultCommandTimeoutSeconds) * time.Second,
		Environment:           cloneStrings(input.Environment),
		Repository:            repository,
		SnapshotID:            snapshotID,
	}, nil
}

func parseRepository(input *contracts.RepositoryConfig) (*sandbox.RepositoryConfig, error) {
	if input == nil {
		return nil, nil
	}
	if input.Provider != "github" || len(input.URL) == 0 || len(input.URL) > 2048 {
		return nil, errors.New("config.repository is invalid")
	}
	parsed, err := url.ParseRequestURI(input.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, errors.New("config.repository.url is invalid")
	}
	if len(input.Path) == 0 || len(input.Path) > 4096 || (input.Path != "/workspace" && !strings.HasPrefix(input.Path, "/workspace/")) {
		return nil, errors.New("config.repository.path is invalid")
	}
	if input.Ref != nil && (len(*input.Ref) == 0 || len(*input.Ref) > 256) {
		return nil, errors.New("config.repository.ref is invalid")
	}
	ref := ""
	if input.Ref != nil {
		ref = *input.Ref
	}
	return &sandbox.RepositoryConfig{Provider: input.Provider, URL: input.URL, Ref: ref, Path: input.Path}, nil
}

func queryLimit(value string) (int, error) {
	if value == "" {
		return defaultListLimit, nil
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit < 1 || limit > maxListLimit {
		return 0, errors.New("limit must be between 1 and 100")
	}
	return limit, nil
}

func parseSandboxID(value string) (sandbox.SandboxID, bool) {
	return sandbox.SandboxID(value), sandboxIDPattern.MatchString(value)
}

func parseSnapshotID(value string) (sandbox.SnapshotID, bool) {
	return sandbox.SnapshotID(value), snapshotIDPattern.MatchString(value)
}

func validState(value sandbox.State) bool {
	switch value {
	case sandbox.StateRequested, sandbox.StateProvisioning, sandbox.StateStarting, sandbox.StateRunning, sandbox.StateSnapshotting, sandbox.StateStopping, sandbox.StateStopped, sandbox.StateFailed, sandbox.StateDestroyed:
		return true
	default:
		return false
	}
}

func cloneStrings(values map[string]string) map[string]string {
	if values == nil {
		return map[string]string{}
	}
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

func toContractSandbox(value sandbox.Sandbox) contracts.Sandbox {
	config := contracts.SandboxConfig{
		Image:                        value.Config.Image,
		CpuMillis:                    int32(value.Config.CPUMillis),
		MemoryMiB:                    int32(value.Config.MemoryMiB),
		StorageGiB:                   int32(value.Config.StorageGiB),
		MaxLifetimeSeconds:           int32(value.Config.MaxLifetime / time.Second),
		DefaultCommandTimeoutSeconds: int32(value.Config.DefaultCommandTimeout / time.Second),
		Environment:                  cloneStrings(value.Config.Environment),
		Repository:                   toContractRepository(value.Repository),
	}
	if value.Config.SnapshotID != nil {
		snapshotID := contracts.SnapshotID(*value.Config.SnapshotID)
		config.SnapshotID = &snapshotID
	}
	var currentCommand *contracts.CommandID
	if value.CurrentCommand != nil {
		commandID := contracts.CommandID(*value.CurrentCommand)
		currentCommand = &commandID
	}
	return contracts.Sandbox{
		ID:               contracts.SandboxID(value.ID),
		State:            contracts.SandboxState(value.State),
		Repository:       toContractRepository(value.Repository),
		Config:           config,
		CreatedAt:        value.CreatedAt.UTC(),
		ExpiresAt:        value.ExpiresAt.UTC(),
		LastActivityAt:   value.LastActivityAt.UTC(),
		CurrentCommandID: currentCommand,
		SnapshotIds:      toContractSnapshotIDs(value.SnapshotIDs),
	}
}

func toContractRepository(value *sandbox.RepositoryConfig) *contracts.RepositoryConfig {
	if value == nil {
		return nil
	}
	ref := contracts.RepositoryConfig{Provider: value.Provider, URL: value.URL, Path: value.Path}
	if value.Ref != "" {
		refValue := value.Ref
		ref.Ref = &refValue
	}
	return &ref
}

func toContractSnapshotIDs(values []sandbox.SnapshotID) []contracts.SnapshotID {
	ids := make([]contracts.SnapshotID, len(values))
	for i, value := range values {
		ids[i] = contracts.SnapshotID(value)
	}
	return ids
}

func sandboxPath(id sandbox.SandboxID) string { return "/v1/sandboxes/" + url.PathEscape(string(id)) }

func snapshotPath(id sandbox.SnapshotID) string { return "/v1/snapshots/" + url.PathEscape(string(id)) }

func toContractSnapshot(value sandbox.SnapshotMetadata) contracts.SnapshotMetadata {
	result := contracts.SnapshotMetadata{
		ID:        contracts.SnapshotID(value.ID),
		SandboxID: contracts.SandboxID(value.SandboxID),
		State:     value.State,
		ObjectKey: value.ObjectKey,
		CreatedAt: value.CreatedAt.UTC(),
	}
	if value.ByteSize >= 0 {
		byteSize := int32(value.ByteSize)
		result.ByteSize = &byteSize
	}
	if value.SHA256 != "" {
		checksum := value.SHA256
		result.Checksum = &checksum
	}
	if value.ExpiresAt != nil {
		expiresAt := value.ExpiresAt.UTC()
		result.ExpiresAt = &expiresAt
	}
	return result
}

func (server *Server) writeServiceError(writer http.ResponseWriter, request *http.Request, err error) {
	status, code, message, details := serviceError(err)
	server.writeError(writer, request, status, code, message, details)
}

func serviceError(err error) (int, string, string, map[string]any) {
	if errors.Is(err, sandbox.ErrNotFound) {
		return http.StatusNotFound, "not_found", "Sandbox was not found.", nil
	}
	if errors.Is(err, sandbox.ErrNotRunning) {
		return http.StatusConflict, "state_conflict", "Sandbox is not running.", nil
	}
	if errors.Is(err, sandbox.ErrSnapshotChecksumMismatch) {
		return http.StatusConflict, "snapshot_checksum_mismatch", "Snapshot archive verification failed.", nil
	}
	if errors.Is(err, sandbox.ErrSnapshotExpired) {
		return http.StatusConflict, "snapshot_expired", "Snapshot metadata has expired.", nil
	}
	if errors.Is(err, sandbox.ErrSnapshotUnavailable) {
		return http.StatusConflict, "snapshot_unavailable", "Snapshot is not available.", nil
	}
	var transition sandbox.InvalidStateTransition
	if errors.As(err, &transition) {
		return http.StatusConflict, "state_conflict", "Sandbox is in a state that does not allow this operation.", map[string]any{"current": transition.Current, "requested": transition.Requested}
	}
	return http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil
}

func (server *Server) writeError(writer http.ResponseWriter, request *http.Request, status int, code, message string, details map[string]any) {
	requestID, _ := requestid.FromContext(request.Context())
	if details == nil {
		details = map[string]any{}
	}
	server.writeJSON(writer, status, contracts.ErrorResponse{Error: contracts.ApiError{Code: code, Message: message, RequestID: requestID, Details: details}})
}

func (server *Server) writeJSON(writer http.ResponseWriter, status int, value any) {
	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(value)
}

var errIdempotencyConflict = errors.New("idempotency conflict")

type idempotencyEntry struct {
	fingerprint []byte
	value       sandbox.Sandbox
}

type idempotencyStore struct {
	mu      sync.Mutex
	entries map[string]idempotencyEntry
}

func (store *idempotencyStore) do(ownerID, key string, request []byte, create func() (sandbox.Sandbox, error)) (sandbox.Sandbox, error) {
	fingerprint := sha256.Sum256(request)
	entryKey := ownerID + "\x00" + key
	store.mu.Lock()
	defer store.mu.Unlock()
	if entry, ok := store.entries[entryKey]; ok {
		if string(entry.fingerprint) != string(fingerprint[:]) {
			return sandbox.Sandbox{}, errIdempotencyConflict
		}
		return entry.value, nil
	}
	value, err := create()
	if err != nil {
		return sandbox.Sandbox{}, err
	}
	store.entries[entryKey] = idempotencyEntry{fingerprint: append([]byte(nil), fingerprint[:]...), value: value}
	return value, nil
}

type snapshotIdempotencyEntry struct {
	fingerprint []byte
	value       sandbox.SnapshotMetadata
}

type snapshotIdempotencyStore struct {
	mu      sync.Mutex
	entries map[string]snapshotIdempotencyEntry
}

func (store *snapshotIdempotencyStore) do(ownerID, key string, request []byte, create func() (sandbox.SnapshotMetadata, error)) (sandbox.SnapshotMetadata, error) {
	fingerprint := sha256.Sum256(request)
	entryKey := ownerID + "\x00" + key
	store.mu.Lock()
	defer store.mu.Unlock()
	if entry, ok := store.entries[entryKey]; ok {
		if string(entry.fingerprint) != string(fingerprint[:]) {
			return sandbox.SnapshotMetadata{}, errIdempotencyConflict
		}
		return entry.value, nil
	}
	value, err := create()
	if err != nil {
		return sandbox.SnapshotMetadata{}, err
	}
	store.entries[entryKey] = snapshotIdempotencyEntry{fingerprint: append([]byte(nil), fingerprint[:]...), value: value}
	return value, nil
}

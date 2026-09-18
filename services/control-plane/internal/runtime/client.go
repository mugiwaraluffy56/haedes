package runtime

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

const maxArchiveBytes = 100 << 20

// Client is the control-plane adapter for the authenticated runtime HTTP API.
// It deliberately owns no sandbox state; command metadata is retained only
// long enough to enrich a reconnecting event stream.
type Client struct {
	httpClient *http.Client
	commands   sync.Map
}

type commandMetadata struct {
	sandboxID sandbox.SandboxID
	command   string
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 2 * time.Minute}
	}
	return &Client{httpClient: httpClient}
}

func (client *Client) WaitReady(ctx context.Context, endpoint, token string) error {
	for {
		response, err := client.do(ctx, http.MethodGet, endpoint, "/healthz", token, nil, "")
		if err == nil {
			statusErr := checkStatus(response)
			response.Body.Close()
			if statusErr == nil {
				return nil
			}
			err = statusErr
		}
		if ctx.Err() != nil {
			return fmt.Errorf("runtime did not become ready: %w", err)
		}
		timer := time.NewTimer(250 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return fmt.Errorf("runtime did not become ready: %w", ctx.Err())
		case <-timer.C:
		}
	}
}

func (client *Client) StartCommand(ctx context.Context, endpoint, token string, request sandbox.CommandRequest) (sandbox.CommandResult, error) {
	timeoutSeconds := uint64(request.Timeout / time.Second)
	if timeoutSeconds == 0 {
		timeoutSeconds = 1
	}
	payload := struct {
		Command        string            `json:"command"`
		CWD            string            `json:"cwd,omitempty"`
		Environment    map[string]string `json:"environment,omitempty"`
		TimeoutSeconds uint64            `json:"timeoutSeconds"`
	}{request.Command, request.CWD, request.Environment, timeoutSeconds}
	body, err := json.Marshal(payload)
	if err != nil {
		return sandbox.CommandResult{}, err
	}
	started := time.Now().UTC()
	response, err := client.do(ctx, http.MethodPost, endpoint, "/v1/commands", token, bytes.NewReader(body), "application/json")
	if err != nil {
		return sandbox.CommandResult{}, err
	}
	defer response.Body.Close()
	if err := checkStatus(response); err != nil {
		return sandbox.CommandResult{}, err
	}
	var payloadResponse struct {
		ID        sandbox.CommandID `json:"id"`
		SandboxID sandbox.SandboxID `json:"sandboxId"`
		Command   string            `json:"command"`
		ExitCode  *int              `json:"exitCode"`
		Signal    string            `json:"signal"`
		TimedOut  bool              `json:"timedOut"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payloadResponse); err != nil {
		return sandbox.CommandResult{}, fmt.Errorf("decode runtime command response: %w", err)
	}
	finished := time.Now().UTC()
	result := sandbox.CommandResult{
		ID: payloadResponse.ID, SandboxID: payloadResponse.SandboxID,
		Command: payloadResponse.Command, ExitCode: payloadResponse.ExitCode,
		Signal: payloadResponse.Signal, TimedOut: payloadResponse.TimedOut,
		StartedAt: started, FinishedAt: &finished,
	}
	if result.Command == "" {
		result.Command = request.Command
	}
	client.commands.Store(result.ID, commandMetadata{sandboxID: result.SandboxID, command: result.Command})
	return result, nil
}

func (client *Client) Events(ctx context.Context, endpoint, token string, commandID sandbox.CommandID, lastEventID string) (<-chan sandbox.CommandEvent, error) {
	path := "/v1/commands/" + url.PathEscape(string(commandID)) + "/events"
	headers := map[string]string{}
	if lastEventID != "" {
		headers["Last-Event-ID"] = lastEventID
	}
	response, err := client.doWithHeaders(ctx, http.MethodGet, endpoint, path, token, nil, "", headers)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(response); err != nil {
		response.Body.Close()
		return nil, err
	}
	metadataValue, _ := client.commands.Load(commandID)
	metadata, _ := metadataValue.(commandMetadata)
	result := make(chan sandbox.CommandEvent)
	go func() {
		defer close(result)
		defer response.Body.Close()
		client.readEvents(ctx, response.Body, commandID, metadata, result)
	}()
	return result, nil
}

func (client *Client) readEvents(ctx context.Context, body io.Reader, commandID sandbox.CommandID, metadata commandMetadata, output chan<- sandbox.CommandEvent) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 4096), 2<<20)
	var eventID, eventType, data string
	emit := func() bool {
		if data == "" {
			return true
		}
		var payload struct {
			Data   string `json:"data"`
			Result *struct {
				ExitCode *int   `json:"exitCode"`
				Signal   string `json:"signal"`
				TimedOut bool   `json:"timedOut"`
			} `json:"result"`
		}
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			return true
		}
		sequence, _ := strconv.ParseInt(eventID, 10, 64)
		event := sandbox.CommandEvent{Sequence: sequence, Type: eventType, CommandID: commandID, Data: payload.Data, At: time.Now().UTC()}
		if payload.Result != nil {
			event.Result = &sandbox.CommandResult{
				ID: commandID, SandboxID: metadata.sandboxID, Command: metadata.command,
				ExitCode: payload.Result.ExitCode, Signal: payload.Result.Signal, TimedOut: payload.Result.TimedOut,
				StartedAt: time.Now().UTC(),
			}
		}
		select {
		case output <- event:
			return true
		case <-ctx.Done():
			return false
		}
	}
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if !emit() {
				return
			}
			eventID, eventType, data = "", "", ""
		case strings.HasPrefix(line, "id:"):
			eventID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
		case strings.HasPrefix(line, "event:"):
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			if data != "" {
				data += "\n"
			}
			data += strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
	}
	if data != "" {
		_ = emit()
	}
}

func (client *Client) ReadFile(ctx context.Context, endpoint, token, path string) (io.ReadCloser, error) {
	response, err := client.fileRequest(ctx, http.MethodGet, endpoint, token, path, nil)
	if err != nil {
		return nil, err
	}
	if err := checkStatus(response); err != nil {
		response.Body.Close()
		return nil, err
	}
	return response.Body, nil
}

func (client *Client) WriteFile(ctx context.Context, endpoint, token, path string, body io.Reader) error {
	response, err := client.fileRequest(ctx, http.MethodPut, endpoint, token, path, body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return checkStatus(response)
}

func (client *Client) ListFiles(ctx context.Context, endpoint, token, path string) ([]sandbox.FileEntry, error) {
	response, err := client.do(ctx, http.MethodGet, endpoint, "/v1/files?path="+url.QueryEscape(path), token, nil, "")
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if err := checkStatus(response); err != nil {
		return nil, err
	}
	var entries []struct {
		Path     string `json:"path"`
		Kind     string `json:"kind"`
		ByteSize int64  `json:"byteSize"`
	}
	if err := json.NewDecoder(response.Body).Decode(&entries); err != nil {
		return nil, err
	}
	result := make([]sandbox.FileEntry, 0, len(entries))
	for _, entry := range entries {
		result = append(result, sandbox.FileEntry{Path: entry.Path, Kind: entry.Kind, ByteSize: entry.ByteSize})
	}
	return result, nil
}

func (client *Client) DeleteFile(ctx context.Context, endpoint, token, path string) error {
	response, err := client.fileRequest(ctx, http.MethodDelete, endpoint, token, path, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return checkStatus(response)
}

func (client *Client) ExportWorkspace(ctx context.Context, endpoint, token string) (io.ReadCloser, sandbox.ArchiveInfo, error) {
	response, err := client.do(ctx, http.MethodGet, endpoint, "/v1/snapshot/export", token, nil, "")
	if err != nil {
		return nil, sandbox.ArchiveInfo{}, err
	}
	if err := checkStatus(response); err != nil {
		response.Body.Close()
		return nil, sandbox.ArchiveInfo{}, err
	}
	byteSize, _ := strconv.ParseInt(response.Header.Get("Content-Length"), 10, 64)
	return response.Body, sandbox.ArchiveInfo{ByteSize: byteSize, SHA256: response.Header.Get("X-Haedes-SHA256"), MediaType: response.Header.Get("Content-Type")}, nil
}

func (client *Client) RestoreWorkspace(ctx context.Context, endpoint, token string, archive io.Reader) error {
	body, err := io.ReadAll(io.LimitReader(archive, maxArchiveBytes+1))
	if err != nil {
		return err
	}
	if len(body) > maxArchiveBytes {
		return fmt.Errorf("runtime archive exceeds %d bytes", maxArchiveBytes)
	}
	digest := sha256.Sum256(body)
	response, err := client.doWithHeaders(ctx, http.MethodPut, endpoint, "/v1/snapshot/restore", token, bytes.NewReader(body), "application/octet-stream", map[string]string{
		"X-Haedes-SHA256": hex.EncodeToString(digest[:]),
	})
	if err != nil {
		return err
	}
	defer response.Body.Close()
	return checkStatus(response)
}

func (client *Client) fileRequest(ctx context.Context, method, endpoint, token, path string, body io.Reader) (*http.Response, error) {
	return client.do(ctx, method, endpoint, "/v1/files/content?path="+url.QueryEscape(path), token, body, "application/octet-stream")
}

func (client *Client) do(ctx context.Context, method, endpoint, path, token string, body io.Reader, contentType string) (*http.Response, error) {
	return client.doWithHeaders(ctx, method, endpoint, path, token, body, contentType, nil)
}

func (client *Client) doWithHeaders(ctx context.Context, method, endpoint, path, token string, body io.Reader, contentType string, headers map[string]string) (*http.Response, error) {
	base := strings.TrimRight(endpoint, "/")
	request, err := http.NewRequestWithContext(ctx, method, base+path, body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Authorization", "Bearer "+token)
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	return client.httpClient.Do(request)
}

func checkStatus(response *http.Response) error {
	if response.StatusCode >= 200 && response.StatusCode < 300 {
		return nil
	}
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
	return fmt.Errorf("runtime HTTP %s: %s", response.Status, strings.TrimSpace(string(body)))
}

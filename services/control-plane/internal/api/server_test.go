package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/api"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/auth"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/contracts"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/fakes"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

func TestCreateSandboxReturnsAcceptedResourceAndSupportsIdempotentRetry(t *testing.T) {
	server, repository, compute := testServer(t, "owner-1")
	body := validCreateBody(t, "image:test")
	request := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-api-key")
	request.Header.Set("Idempotency-Key", "create-1")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if recorder.Header().Get("Location") != "/v1/sandboxes/sbx_001" || !strings.HasPrefix(recorder.Header().Get("X-Request-ID"), "req_") {
		t.Fatalf("headers = %#v", recorder.Header())
	}
	var created contracts.Sandbox
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID != "sbx_001" || created.State != contracts.SandboxStateRunning || created.Config.Image != "image:test" {
		t.Fatalf("created sandbox = %+v", created)
	}

	retry := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewReader(body))
	retry.Header.Set("Authorization", "Bearer test-api-key")
	retry.Header.Set("Idempotency-Key", "create-1")
	retryRecorder := httptest.NewRecorder()
	server.ServeHTTP(retryRecorder, retry)
	if retryRecorder.Code != http.StatusAccepted {
		t.Fatalf("retry status = %d, body = %s", retryRecorder.Code, retryRecorder.Body.String())
	}
	if len(compute.Events) != 1 || len(repository.Updates) != 3 {
		t.Fatalf("retry created another sandbox: compute=%+v updates=%+v", compute.Events, repository.Updates)
	}
}

func TestCreateRejectsMalformedInputLimitsAndIdempotencyReuse(t *testing.T) {
	server, _, _ := testServer(t, "owner-1")
	tests := []struct {
		name   string
		body   string
		header string
	}{
		{name: "missing idempotency", body: "{}", header: ""},
		{name: "invalid limit", body: string(validCreateBody(t, "image:test")), header: "bad key"},
		{name: "unknown field", body: `{"config":{},"extra":true}`, header: "create-2"},
		{name: "invalid config", body: `{"config":{"image":"image:test","cpuMillis":1}}`, header: "create-3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", strings.NewReader(test.body))
			request.Header.Set("Authorization", "Bearer test-api-key")
			if test.header != "" {
				request.Header.Set("Idempotency-Key", test.header)
			}
			recorder := httptest.NewRecorder()
			server.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			assertErrorCode(t, recorder, "invalid_request")
		})
	}

	body := validCreateBody(t, "image:one")
	first := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewReader(body))
	first.Header.Set("Authorization", "Bearer test-api-key")
	first.Header.Set("Idempotency-Key", "same-key")
	server.ServeHTTP(httptest.NewRecorder(), first)
	second := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewReader(validCreateBody(t, "image:two")))
	second.Header.Set("Authorization", "Bearer test-api-key")
	second.Header.Set("Idempotency-Key", "same-key")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, second)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	assertErrorCode(t, recorder, "idempotency_conflict")
}

func TestLifecycleRoutesRequireAuthAndRespectOwnerScope(t *testing.T) {
	server, _, _ := testServer(t, "owner-1")
	unauthorized := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_missing", nil)
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, unauthorized)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", recorder.Code)
	}
	assertErrorCode(t, recorder, "authentication_required")

	body := validCreateBody(t, "image:test")
	create := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewReader(body))
	create.Header.Set("Authorization", "Bearer test-api-key")
	create.Header.Set("Idempotency-Key", "owner-scope")
	server.ServeHTTP(httptest.NewRecorder(), create)

	otherServer, _, _ := testServer(t, "owner-2")
	get := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001", nil)
	get.Header.Set("Authorization", "Bearer test-api-key")
	getRecorder := httptest.NewRecorder()
	otherServer.ServeHTTP(getRecorder, get)
	if getRecorder.Code != http.StatusNotFound {
		t.Fatalf("cross-owner get status = %d, body = %s", getRecorder.Code, getRecorder.Body.String())
	}
	assertErrorCode(t, getRecorder, "not_found")
}

func TestListAndDestroyRoutesReturnDocumentedResponses(t *testing.T) {
	server, _, _ := testServer(t, "owner-1")
	body := validCreateBody(t, "image:test")
	create := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewReader(body))
	create.Header.Set("Authorization", "Bearer test-api-key")
	create.Header.Set("Idempotency-Key", "list-destroy")
	server.ServeHTTP(httptest.NewRecorder(), create)

	list := httptest.NewRequest(http.MethodGet, "/v1/sandboxes?limit=1&state=running", nil)
	list.Header.Set("Authorization", "Bearer test-api-key")
	listRecorder := httptest.NewRecorder()
	server.ServeHTTP(listRecorder, list)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var page contracts.SandboxPage
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "sbx_001" {
		t.Fatalf("list page = %+v", page)
	}

	destroy := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/sbx_001", nil)
	destroy.Header.Set("Authorization", "Bearer test-api-key")
	destroyRecorder := httptest.NewRecorder()
	server.ServeHTTP(destroyRecorder, destroy)
	if destroyRecorder.Code != http.StatusAccepted || destroyRecorder.Header().Get("Location") != "/v1/sandboxes/sbx_001" {
		t.Fatalf("destroy status = %d, headers = %#v", destroyRecorder.Code, destroyRecorder.Header())
	}
	repeated := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/sbx_001", nil)
	repeated.Header.Set("Authorization", "Bearer test-api-key")
	repeatedRecorder := httptest.NewRecorder()
	server.ServeHTTP(repeatedRecorder, repeated)
	if repeatedRecorder.Code != http.StatusAccepted {
		t.Fatalf("repeated destroy status = %d, body = %s", repeatedRecorder.Code, repeatedRecorder.Body.String())
	}
}

func testServer(t *testing.T, ownerID string) (*api.Server, *fakes.Repository, *fakes.Compute) {
	t.Helper()
	clock := fakes.NewClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	compute := fakes.NewCompute(clock)
	compute.StartTask = sandbox.TaskRef{ARN: "task-fixed", Endpoint: "http://runtime/fixed", RuntimeToken: "runtime-token"}
	runtime := fakes.NewRuntime(clock)
	runtime.Register("sbx_001", compute.StartTask)
	repository := fakes.NewRepository()
	service := sandbox.NewService(sandbox.Dependencies{Repository: repository, Compute: compute, Runtime: runtime, IDs: testIDs{}, Clock: clock})
	record, err := auth.NewAPIKeyRecord("key-test", ownerID, "test-api-key")
	if err != nil {
		t.Fatal(err)
	}
	store, err := auth.NewMemoryStore(record)
	if err != nil {
		t.Fatal(err)
	}
	authentication, err := auth.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	return api.NewServer(service, authentication), repository, compute
}

func validCreateBody(t *testing.T, image string) []byte {
	t.Helper()
	body, err := json.Marshal(contracts.SandboxCreateRequest{Config: contracts.SandboxConfig{
		Image:                        image,
		CpuMillis:                    512,
		MemoryMiB:                    1024,
		StorageGiB:                   10,
		MaxLifetimeSeconds:           600,
		DefaultCommandTimeoutSeconds: 30,
		Environment:                  map[string]string{"MODE": "test"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func assertErrorCode(t *testing.T, recorder *httptest.ResponseRecorder, expected string) {
	t.Helper()
	var response contracts.ErrorResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != expected || !strings.HasPrefix(response.Error.RequestID, "req_") {
		t.Fatalf("error = %+v", response.Error)
	}
}

type testIDs struct{}

func (testIDs) NewSandboxID() sandbox.SandboxID   { return "sbx_001" }
func (testIDs) NewCommandID() sandbox.CommandID   { return "cmd_001" }
func (testIDs) NewSnapshotID() sandbox.SnapshotID { return "snp_001" }
func (testIDs) NewRuntimeToken() string           { return "runtime-token" }

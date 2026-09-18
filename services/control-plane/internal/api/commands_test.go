package api_test

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/api"
)

func TestCommandSubmissionReturnsAcceptedResultAndReplayableSSE(t *testing.T) {
	server, _, _, _ := testServerWithRuntime(t, "owner-1")
	createSandboxForCommand(t, server)
	commandBody := `{"command":"printf ok","cwd":"/workspace","environment":{"MODE":"test"},"timeoutSeconds":5}`
	request := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/sbx_001/commands", strings.NewReader(commandBody))
	request.Header.Set("Authorization", "Bearer test-api-key")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted || recorder.Header().Get("Location") != "/v1/sandboxes/sbx_001/commands/cmd_001" {
		t.Fatalf("command response: status=%d headers=%#v body=%s", recorder.Code, recorder.Header(), recorder.Body.String())
	}

	eventsRequest := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/commands/cmd_001/events", nil)
	eventsRequest.Header.Set("Authorization", "Bearer test-api-key")
	eventsRecorder := httptest.NewRecorder()
	server.ServeHTTP(eventsRecorder, eventsRequest)
	eventsBody := eventsRecorder.Body.String()
	if eventsRecorder.Code != http.StatusOK || eventsRecorder.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("SSE response: status=%d headers=%#v body=%s", eventsRecorder.Code, eventsRecorder.Header(), eventsBody)
	}
	if !strings.Contains(eventsBody, "id: 1\nevent: started") || !strings.Contains(eventsBody, "id: 2\nevent: completed") {
		t.Fatalf("SSE body = %s", eventsBody)
	}

	replayRequest := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/commands/cmd_001/events", nil)
	replayRequest.Header.Set("Authorization", "Bearer test-api-key")
	replayRequest.Header.Set("Last-Event-ID", "1")
	replayRecorder := httptest.NewRecorder()
	server.ServeHTTP(replayRecorder, replayRequest)
	if strings.Contains(replayRecorder.Body.String(), "event: started") || !strings.Contains(replayRecorder.Body.String(), "event: completed") {
		t.Fatalf("replayed SSE body = %s", replayRecorder.Body.String())
	}
}

func TestCommandRoutesValidateInputAndMapMissingOrStoppedCommands(t *testing.T) {
	server, _, _, _ := testServerWithRuntime(t, "owner-1")
	createSandboxForCommand(t, server)
	invalid := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/sbx_001/commands", strings.NewReader(`{"command":"","timeoutSeconds":901}`))
	invalid.Header.Set("Authorization", "Bearer test-api-key")
	invalidRecorder := httptest.NewRecorder()
	server.ServeHTTP(invalidRecorder, invalid)
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("invalid command status = %d, body = %s", invalidRecorder.Code, invalidRecorder.Body.String())
	}

	missing := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/commands/cmd_missing/events", nil)
	missing.Header.Set("Authorization", "Bearer test-api-key")
	missingRecorder := httptest.NewRecorder()
	server.ServeHTTP(missingRecorder, missing)
	if missingRecorder.Code != http.StatusNotFound || !strings.Contains(missingRecorder.Body.String(), "command_not_found") {
		t.Fatalf("missing command response: status=%d body=%s", missingRecorder.Code, missingRecorder.Body.String())
	}

	destroy := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/sbx_001", nil)
	destroy.Header.Set("Authorization", "Bearer test-api-key")
	server.ServeHTTP(httptest.NewRecorder(), destroy)
	stopped := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/sbx_001/commands", strings.NewReader(`{"command":"printf stopped"}`))
	stopped.Header.Set("Authorization", "Bearer test-api-key")
	stopped.Header.Set("Content-Type", "application/json")
	stoppedRecorder := httptest.NewRecorder()
	server.ServeHTTP(stoppedRecorder, stopped)
	if stoppedRecorder.Code != http.StatusConflict || !strings.Contains(stoppedRecorder.Body.String(), "state_conflict") {
		t.Fatalf("stopped command response: status=%d body=%s", stoppedRecorder.Code, stoppedRecorder.Body.String())
	}
}

func TestSSEHeartbeatsContinueUntilClientDisconnects(t *testing.T) {
	server, _, _, _ := testServerWithRuntime(t, "owner-1")
	createSandboxForCommand(t, server)
	command := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/sbx_001/commands", strings.NewReader(`{"command":"printf heartbeat"}`))
	command.Header.Set("Authorization", "Bearer test-api-key")
	server.ServeHTTP(httptest.NewRecorder(), command)

	streamServer := server.WithHeartbeat(time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	request := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/commands/cmd_001/events", nil).WithContext(ctx)
	request.Header.Set("Authorization", "Bearer test-api-key")
	request.Header.Set("Last-Event-ID", "2")
	recorder := httptest.NewRecorder()
	done := make(chan struct{})
	go func() {
		streamServer.ServeHTTP(recorder, request)
		close(done)
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE stream did not stop after client disconnect")
	}
	if !strings.Contains(recorder.Body.String(), ": heartbeat") {
		t.Fatalf("heartbeat body = %s", recorder.Body.String())
	}
}

func TestCommandRuntimeFailureAndInvalidLastEventID(t *testing.T) {
	server, _, _, runtime := testServerWithRuntime(t, "owner-1")
	createSandboxForCommand(t, server)
	runtime.StartCommandError = errors.New("runtime unavailable")
	request := httptest.NewRequest(http.MethodPost, "/v1/sandboxes/sbx_001/commands", strings.NewReader(`{"command":"printf fail"}`))
	request.Header.Set("Authorization", "Bearer test-api-key")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), "runtime_failure") {
		t.Fatalf("runtime failure response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}

	invalidEventID := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/commands/cmd_001/events", nil)
	invalidEventID.Header.Set("Authorization", "Bearer test-api-key")
	invalidEventID.Header.Set("Last-Event-ID", "not-a-number")
	invalidRecorder := httptest.NewRecorder()
	server.ServeHTTP(invalidRecorder, invalidEventID)
	if invalidRecorder.Code != http.StatusBadRequest || !strings.Contains(invalidRecorder.Body.String(), "invalid_request") {
		t.Fatalf("invalid event ID response: status=%d body=%s", invalidRecorder.Code, invalidRecorder.Body.String())
	}
}

func createSandboxForCommand(t *testing.T, server *api.Server) {
	t.Helper()
	body := validCreateBody(t, "image:command")
	request := httptest.NewRequest(http.MethodPost, "/v1/sandboxes", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-api-key")
	request.Header.Set("Idempotency-Key", "command-sandbox")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("sandbox setup status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

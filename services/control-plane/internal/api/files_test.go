package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/contracts"
)

func TestFilesystemRoutesDelegateReadWriteListAndDelete(t *testing.T) {
	server, _, _, runtime := testServerWithRuntime(t, "owner-1")
	createSandboxForCommand(t, server)

	write := httptest.NewRequest(http.MethodPut, "/v1/sandboxes/sbx_001/files/content?path=/workspace/hello.txt", strings.NewReader("hello"))
	write.Header.Set("Authorization", "Bearer test-api-key")
	writeRecorder := httptest.NewRecorder()
	server.ServeHTTP(writeRecorder, write)
	if writeRecorder.Code != http.StatusOK {
		t.Fatalf("write status = %d, body = %s", writeRecorder.Code, writeRecorder.Body.String())
	}
	var written contracts.FileEntry
	if err := json.Unmarshal(writeRecorder.Body.Bytes(), &written); err != nil {
		t.Fatal(err)
	}
	if written.Path != "/workspace/hello.txt" || written.Kind != "file" || written.ByteSize == nil || *written.ByteSize != 5 {
		t.Fatalf("written entry = %+v", written)
	}

	read := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/files/content?path=/workspace/hello.txt", nil)
	read.Header.Set("Authorization", "Bearer test-api-key")
	readRecorder := httptest.NewRecorder()
	server.ServeHTTP(readRecorder, read)
	if readRecorder.Code != http.StatusOK || readRecorder.Body.String() != "hello" || readRecorder.Header().Get("ETag") == "" {
		t.Fatalf("read response: status=%d headers=%#v body=%q", readRecorder.Code, readRecorder.Header(), readRecorder.Body.String())
	}

	list := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/files?path=/workspace", nil)
	list.Header.Set("Authorization", "Bearer test-api-key")
	listRecorder := httptest.NewRecorder()
	server.ServeHTTP(listRecorder, list)
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("list status = %d, body = %s", listRecorder.Code, listRecorder.Body.String())
	}
	var listing contracts.FileListResponse
	if err := json.Unmarshal(listRecorder.Body.Bytes(), &listing); err != nil {
		t.Fatal(err)
	}
	if listing.Path != "/workspace" || len(listing.Entries) != 1 || listing.Entries[0].Path != "/workspace/hello.txt" {
		t.Fatalf("listing = %+v", listing)
	}

	deleteRequest := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/sbx_001/files/content?path=/workspace/hello.txt", nil)
	deleteRequest.Header.Set("Authorization", "Bearer test-api-key")
	deleteRecorder := httptest.NewRecorder()
	server.ServeHTTP(deleteRecorder, deleteRequest)
	if deleteRecorder.Code != http.StatusNoContent || deleteRecorder.Body.Len() != 0 {
		t.Fatalf("delete response: status=%d body=%q", deleteRecorder.Code, deleteRecorder.Body.String())
	}

	missing := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/files/content?path=/workspace/hello.txt", nil)
	missing.Header.Set("Authorization", "Bearer test-api-key")
	missingRecorder := httptest.NewRecorder()
	server.ServeHTTP(missingRecorder, missing)
	if missingRecorder.Code != http.StatusNotFound || !strings.Contains(missingRecorder.Body.String(), "file_not_found") {
		t.Fatalf("missing file response: status=%d body=%s", missingRecorder.Code, missingRecorder.Body.String())
	}

	if len(runtime.RuntimeEvents) < 5 {
		t.Fatalf("runtime events = %+v", runtime.RuntimeEvents)
	}
}

func TestFilesystemRoutesRejectTraversalUnauthorizedAndStoppedAccess(t *testing.T) {
	server, _, _, _ := testServerWithRuntime(t, "owner-1")
	createSandboxForCommand(t, server)
	paths := []string{"/workspace/../etc/passwd", "/etc/passwd", "/workspace/."}
	for _, path := range paths {
		request := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/files/content?path="+path, nil)
		request.Header.Set("Authorization", "Bearer test-api-key")
		recorder := httptest.NewRecorder()
		server.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("path %q status = %d, body = %s", path, recorder.Code, recorder.Body.String())
		}
	}

	unauthorized := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/files", nil)
	unauthorizedRecorder := httptest.NewRecorder()
	server.ServeHTTP(unauthorizedRecorder, unauthorized)
	if unauthorizedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", unauthorizedRecorder.Code)
	}

	destroy := httptest.NewRequest(http.MethodDelete, "/v1/sandboxes/sbx_001", nil)
	destroy.Header.Set("Authorization", "Bearer test-api-key")
	server.ServeHTTP(httptest.NewRecorder(), destroy)
	stopped := httptest.NewRequest(http.MethodGet, "/v1/sandboxes/sbx_001/files/content?path=/workspace/file.txt", nil)
	stopped.Header.Set("Authorization", "Bearer test-api-key")
	stoppedRecorder := httptest.NewRecorder()
	server.ServeHTTP(stoppedRecorder, stopped)
	if stoppedRecorder.Code != http.StatusConflict || !strings.Contains(stoppedRecorder.Body.String(), "state_conflict") {
		t.Fatalf("stopped response: status=%d body=%s", stoppedRecorder.Code, stoppedRecorder.Body.String())
	}
}

func TestFilesystemWriteEnforcesBodyLimit(t *testing.T) {
	server, _, _, _ := testServerWithRuntime(t, "owner-1")
	createSandboxForCommand(t, server)
	body := bytes.Repeat([]byte("x"), 10*1024*1024+1)
	request := httptest.NewRequest(http.MethodPut, "/v1/sandboxes/sbx_001/files/content?path=/workspace/large.txt", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer test-api-key")
	recorder := httptest.NewRecorder()
	server.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusRequestEntityTooLarge || !strings.Contains(recorder.Body.String(), "payload_too_large") {
		t.Fatalf("large write response: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

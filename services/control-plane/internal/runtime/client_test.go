package runtime

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

func TestClientCoversRuntimeSmokeSurface(t *testing.T) {
	handler := http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get("Authorization") != "Bearer token" {
			http.Error(writer, "unauthorized", http.StatusUnauthorized)
			return
		}
		switch {
		case request.Method == http.MethodGet && request.URL.Path == "/healthz":
			writer.WriteHeader(http.StatusOK)
		case request.Method == http.MethodPost && request.URL.Path == "/v1/commands":
			_, _ = writer.Write([]byte(`{"id":"cmd_1","sandboxId":"sbx_1","command":"printf ok","exitCode":0,"timedOut":false}`))
		case request.Method == http.MethodGet && request.URL.Path == "/v1/commands/cmd_1/events":
			writer.Header().Set("Content-Type", "text/event-stream")
			_, _ = writer.Write([]byte("id: 1\nevent: started\ndata: {}\n\nid: 2\nevent: completed\ndata: {\"result\":{\"exitCode\":0,\"timedOut\":false}}\n\n"))
		case request.Method == http.MethodGet && request.URL.Path == "/v1/files":
			_, _ = writer.Write([]byte(`[{"path":"/workspace/smoke.txt","kind":"file","byteSize":4}]`))
		case request.Method == http.MethodGet && request.URL.Path == "/v1/files/content":
			_, _ = writer.Write([]byte("smoke"))
		case request.Method == http.MethodPut && request.URL.Path == "/v1/files/content":
			writer.WriteHeader(http.StatusNoContent)
		case request.Method == http.MethodGet && request.URL.Path == "/v1/snapshot/export":
			writer.Header().Set("Content-Length", "5")
			writer.Header().Set("X-Haedes-SHA256", "abc")
			writer.Header().Set("Content-Type", "application/test")
			_, _ = writer.Write([]byte("test!"))
		default:
			http.NotFound(writer, request)
		}
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("network listeners unavailable: %v", err)
	}
	server := httptest.NewUnstartedServer(handler)
	server.Listener = listener
	server.Start()
	defer server.Close()

	client := NewClient(server.Client())
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := client.WaitReady(ctx, server.URL, "token"); err != nil {
		t.Fatal(err)
	}
	result, err := client.StartCommand(ctx, server.URL, "token", sandbox.CommandRequest{Command: "printf ok", Timeout: time.Second})
	if err != nil || result.ID != "cmd_1" || result.SandboxID != "sbx_1" {
		t.Fatalf("command result = %+v, err = %v", result, err)
	}
	events, err := client.Events(ctx, server.URL, "token", result.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	var completed bool
	for event := range events {
		completed = completed || event.Type == "completed"
	}
	if !completed {
		t.Fatal("runtime event stream did not complete")
	}
	if err := client.WriteFile(ctx, server.URL, "token", "/workspace/smoke.txt", strings.NewReader("smoke")); err != nil {
		t.Fatal(err)
	}
	entries, err := client.ListFiles(ctx, server.URL, "token", "/workspace")
	if err != nil || len(entries) != 1 {
		t.Fatalf("file entries = %+v, err = %v", entries, err)
	}
	content, err := client.ReadFile(ctx, server.URL, "token", "/workspace/smoke.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer content.Close()
	body, _ := io.ReadAll(content)
	if string(body) != "smoke" {
		t.Fatalf("file body = %q", body)
	}
	archive, info, err := client.ExportWorkspace(ctx, server.URL, "token")
	if err != nil {
		t.Fatal(err)
	}
	defer archive.Close()
	if info.ByteSize != 5 || info.SHA256 != "abc" {
		t.Fatalf("archive info = %+v", info)
	}
}

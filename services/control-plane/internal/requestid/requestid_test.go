package requestid

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fixedGenerator string

func (g fixedGenerator) New() string { return string(g) }

func TestMiddlewareGeneratesAndPropagatesRequestID(t *testing.T) {
	var contextID string
	handler := MiddlewareWithGenerator(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		contextID, _ = FromContext(request.Context())
	}), fixedGenerator("req_generated"))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Header().Get(Header) != "req_generated" || contextID != "req_generated" {
		t.Fatalf("request ID was not propagated: header=%q context=%q", recorder.Header().Get(Header), contextID)
	}
}

func TestMiddlewarePreservesValidIncomingRequestID(t *testing.T) {
	handler := MiddlewareWithGenerator(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		id, ok := FromContext(request.Context())
		if !ok || id != "req_client-123" {
			t.Fatalf("unexpected context request ID: %q", id)
		}
	}), fixedGenerator("req_generated"))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(Header, " req_client-123 ")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Header().Get(Header) != "req_client-123" {
		t.Fatalf("incoming request ID was not preserved")
	}
}

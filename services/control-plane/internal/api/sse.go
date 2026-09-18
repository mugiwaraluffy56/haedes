package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

func (server *Server) streamCommandEvents(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, sandboxValue, commandValue string) {
	sandboxID, ok := parseSandboxID(sandboxValue)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	commandID, ok := parseCommandID(commandValue)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Command was not found.", nil)
		return
	}
	lastEventID := request.Header.Get("Last-Event-ID")
	if lastEventID != "" {
		if _, err := strconv.ParseUint(lastEventID, 10, 64); err != nil {
			server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "Last-Event-ID must contain only digits.", nil)
			return
		}
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	streamContext, cancel := context.WithCancel(request.Context())
	defer cancel()
	events, err := server.service.SubscribeEvents(streamContext, principal.OwnerID, sandboxID, commandID, lastEventID)
	if err != nil {
		server.writeCommandError(writer, request, err)
		return
	}
	flusher, ok := writer.(http.Flusher)
	if !ok {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The response does not support streaming.", nil)
		return
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")
	writer.WriteHeader(http.StatusOK)
	flusher.Flush()

	heartbeat := time.NewTicker(server.heartbeatInterval)
	defer heartbeat.Stop()
	for {
		select {
		case event, open := <-events:
			if !open {
				return
			}
			if err := writeSSEEvent(writer, event); err != nil {
				return
			}
			flusher.Flush()
			if event.Type == "completed" || event.Type == "failed" {
				return
			}
		case <-heartbeat.C:
			if _, err := fmt.Fprint(writer, ": heartbeat\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case <-request.Context().Done():
			return
		}
	}
}

func writeSSEEvent(writer http.ResponseWriter, event sandbox.CommandEvent) error {
	payload := map[string]any{
		"type":      event.Type,
		"commandId": string(event.CommandID),
		"at":        event.At.UTC(),
	}
	switch event.Type {
	case "stdout", "stderr":
		payload["data"] = event.Data
	case "completed":
		if event.Result != nil {
			payload["result"] = toContractCommandResult(*event.Result)
		}
	case "failed":
		payload["code"] = "runtime_failure"
		payload["message"] = event.Data
	default:
		if event.Data != "" {
			payload["data"] = event.Data
		}
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "id: %d\nevent: %s\ndata: %s\n\n", event.Sequence, event.Type, data)
	return err
}

func parseCommandID(value string) (sandbox.CommandID, bool) {
	return sandbox.CommandID(value), commandIDPattern.MatchString(value)
}

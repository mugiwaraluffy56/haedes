package api

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/contracts"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

const maxCommandBody = 256 << 10

func (server *Server) executeCommand(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, ok := parseSandboxID(id)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	var input contracts.CommandRequest
	if err := decodeJSON(writer, request, &input, maxCommandBody); err != nil {
		server.writeValidationError(writer, request, err)
		return
	}
	command, err := parseCommandRequest(input)
	if err != nil {
		server.writeValidationError(writer, request, err)
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	result, err := server.service.Execute(request.Context(), principal.OwnerID, sandboxID, command)
	if err != nil {
		server.writeCommandError(writer, request, err)
		return
	}
	writer.Header().Set("Location", commandPath(sandboxID, result.ID))
	server.writeJSON(writer, http.StatusAccepted, toContractCommandResult(result))
}

func parseCommandRequest(input contracts.CommandRequest) (sandbox.CommandRequest, error) {
	if len(input.Command) == 0 {
		return sandbox.CommandRequest{}, errors.New("command must contain 1 to 16384 characters")
	}
	if len(input.Command) > 16384 {
		return sandbox.CommandRequest{}, payloadTooLargeError{message: "command exceeds the 16384 byte limit."}
	}
	cwd := ""
	if input.Cwd != nil {
		cwd = string(*input.Cwd)
		if len(cwd) > 4096 || (cwd != "/workspace" && !strings.HasPrefix(cwd, "/workspace/")) {
			return sandbox.CommandRequest{}, errors.New("cwd must remain beneath /workspace")
		}
	}
	environment := map[string]string{}
	if input.Environment != nil {
		environment = *input.Environment
	}
	if len(environment) > 32 {
		return sandbox.CommandRequest{}, payloadTooLargeError{message: "environment exceeds the 32 entry limit."}
	}
	for key, value := range environment {
		if key == "" {
			return sandbox.CommandRequest{}, errors.New("environment keys and values are invalid")
		}
		if len(value) > 4096 {
			return sandbox.CommandRequest{}, payloadTooLargeError{message: "environment value exceeds the 4096 byte limit."}
		}
	}
	timeout := time.Duration(0)
	if input.TimeoutSeconds != nil {
		if *input.TimeoutSeconds < 1 || *input.TimeoutSeconds > 900 {
			return sandbox.CommandRequest{}, errors.New("timeoutSeconds must be between 1 and 900")
		}
		timeout = time.Duration(*input.TimeoutSeconds) * time.Second
	}
	return sandbox.CommandRequest{Command: input.Command, CWD: cwd, Environment: cloneStrings(environment), Timeout: timeout}, nil
}

func toContractCommandResult(value sandbox.CommandResult) contracts.CommandResult {
	result := contracts.CommandResult{
		ID:        contracts.CommandID(value.ID),
		SandboxID: contracts.SandboxID(value.SandboxID),
		Command:   value.Command,
		StartedAt: value.StartedAt.UTC(),
		TimedOut:  value.TimedOut,
	}
	if value.ExitCode != nil {
		exitCode := int32(*value.ExitCode)
		result.ExitCode = &exitCode
	}
	if value.FinishedAt != nil {
		finishedAt := value.FinishedAt.UTC()
		result.FinishedAt = &finishedAt
	}
	if value.Signal != "" {
		signal := value.Signal
		result.Signal = &signal
	}
	return result
}

func commandPath(sandboxID sandbox.SandboxID, commandID sandbox.CommandID) string {
	return "/v1/sandboxes/" + string(sandboxID) + "/commands/" + string(commandID)
}

func (server *Server) writeCommandError(writer http.ResponseWriter, request *http.Request, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		server.writeError(writer, request, http.StatusRequestTimeout, "command_timeout", "Command exceeded its timeout.", nil)
		return
	}
	if errors.Is(err, sandbox.ErrNotFound) {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	if errors.Is(err, sandbox.ErrCommandNotFound) {
		server.writeError(writer, request, http.StatusNotFound, "command_not_found", "Command was not found.", nil)
		return
	}
	var transition sandbox.InvalidStateTransition
	if errors.Is(err, sandbox.ErrNotRunning) || errors.As(err, &transition) {
		status, code, message, details := serviceError(err)
		server.writeError(writer, request, status, code, message, details)
		return
	}
	server.writeError(writer, request, http.StatusInternalServerError, "runtime_failure", "The sandbox runtime could not complete the command.", nil)
}

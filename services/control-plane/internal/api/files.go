package api

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/contracts"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

const (
	maxFileBytes        = 10 * 1024 * 1024
	maxDirectoryEntries = 10_000
)

func (server *Server) listFiles(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, ok := parseSandboxID(id)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	path := request.URL.Query().Get("path")
	if path == "" {
		path = "/workspace"
	}
	if !validWorkspacePath(path, false) {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "path must remain beneath /workspace.", nil)
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	entries, err := server.service.ListFiles(request.Context(), principal.OwnerID, sandboxID, path)
	if err != nil {
		server.writeFileError(writer, request, err)
		return
	}
	if len(entries) > maxDirectoryEntries {
		server.writeError(writer, request, http.StatusRequestEntityTooLarge, "payload_too_large", "Directory contains too many entries.", nil)
		return
	}
	responseEntries := make([]contracts.FileEntry, 0, len(entries))
	for _, entry := range entries {
		converted, err := toContractFileEntry(entry)
		if err != nil {
			server.writeFileError(writer, request, err)
			return
		}
		responseEntries = append(responseEntries, converted)
	}
	server.writeJSON(writer, http.StatusOK, contracts.FileListResponse{Path: path, Entries: responseEntries})
}

func (server *Server) readFile(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, path, ok := server.fileTarget(writer, request, id)
	if !ok {
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	content, err := server.service.ReadFile(request.Context(), principal.OwnerID, sandboxID, path)
	if err != nil {
		server.writeFileError(writer, request, err)
		return
	}
	defer content.Close()
	body, err := io.ReadAll(io.LimitReader(content, maxFileBytes+1))
	if err != nil {
		server.writeFileError(writer, request, err)
		return
	}
	if len(body) > maxFileBytes {
		server.writeError(writer, request, http.StatusRequestEntityTooLarge, "payload_too_large", "File body exceeds the 10485760 byte limit.", nil)
		return
	}
	digest := sha256.Sum256(body)
	writer.Header().Set("Content-Type", "application/octet-stream")
	writer.Header().Set("ETag", `"sha256-`+hex.EncodeToString(digest[:])+`"`)
	writer.WriteHeader(http.StatusOK)
	_, _ = writer.Write(body)
}

func (server *Server) writeFile(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, path, ok := server.fileTarget(writer, request, id)
	if !ok {
		return
	}
	request.Body = http.MaxBytesReader(writer, request.Body, maxFileBytes+1)
	body, err := io.ReadAll(request.Body)
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			server.writeError(writer, request, http.StatusRequestEntityTooLarge, "payload_too_large", "File body exceeds the 10485760 byte limit.", nil)
			return
		}
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "File body could not be read.", nil)
		return
	}
	if len(body) > maxFileBytes {
		server.writeError(writer, request, http.StatusRequestEntityTooLarge, "payload_too_large", "File body exceeds the 10485760 byte limit.", nil)
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	if err := server.service.WriteFile(request.Context(), principal.OwnerID, sandboxID, path, bytes.NewReader(body)); err != nil {
		server.writeFileError(writer, request, err)
		return
	}
	byteSize := int32(len(body))
	server.writeJSON(writer, http.StatusOK, contracts.FileEntry{Path: path, Kind: "file", ByteSize: &byteSize})
}

func (server *Server) deleteFile(writer http.ResponseWriter, request *http.Request, principal sandbox.Principal, id string) {
	sandboxID, path, ok := server.fileTarget(writer, request, id)
	if !ok {
		return
	}
	if server.service == nil {
		server.writeError(writer, request, http.StatusInternalServerError, "internal_error", "The request could not be completed.", nil)
		return
	}
	if err := server.service.DeleteFile(request.Context(), principal.OwnerID, sandboxID, path); err != nil {
		server.writeFileError(writer, request, err)
		return
	}
	writer.WriteHeader(http.StatusNoContent)
}

func (server *Server) fileTarget(writer http.ResponseWriter, request *http.Request, id string) (sandbox.SandboxID, string, bool) {
	sandboxID, ok := parseSandboxID(id)
	if !ok {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return "", "", false
	}
	path := request.URL.Query().Get("path")
	if !validWorkspacePath(path, true) {
		server.writeError(writer, request, http.StatusBadRequest, "invalid_request", "path must be an absolute file path beneath /workspace.", nil)
		return "", "", false
	}
	return sandboxID, path, true
}

func validWorkspacePath(path string, file bool) bool {
	if len(path) == 0 || len(path) > 4096 || !strings.HasPrefix(path, "/workspace") || strings.ContainsRune(path, '\x00') || strings.ContainsRune(path, '\\') {
		return false
	}
	if file && path == "/workspace" {
		return false
	}
	if path != "/workspace" && !strings.HasPrefix(path, "/workspace/") {
		return false
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "." || segment == ".." {
			return false
		}
	}
	return true
}

func toContractFileEntry(entry sandbox.FileEntry) (contracts.FileEntry, error) {
	if !validWorkspacePath(entry.Path, true) {
		return contracts.FileEntry{}, errors.New("runtime returned an unsafe file path")
	}
	converted := contracts.FileEntry{Path: entry.Path, Kind: entry.Kind}
	if entry.Kind != "file" && entry.Kind != "directory" && entry.Kind != "symlink" {
		return contracts.FileEntry{}, errors.New("runtime returned an invalid file kind")
	}
	if entry.ByteSize < 0 || entry.ByteSize > maxFileBytes {
		return contracts.FileEntry{}, errors.New("runtime returned an invalid file size")
	}
	if entry.ByteSize != 0 {
		size := int32(entry.ByteSize)
		converted.ByteSize = &size
	}
	if !entry.ModifiedAt.IsZero() {
		modified := entry.ModifiedAt.UTC()
		converted.ModifiedAt = &modified
	}
	return converted, nil
}

func (server *Server) writeFileError(writer http.ResponseWriter, request *http.Request, err error) {
	if errors.Is(err, sandbox.ErrNotFound) {
		server.writeError(writer, request, http.StatusNotFound, "not_found", "Sandbox was not found.", nil)
		return
	}
	if errors.Is(err, sandbox.ErrFileNotFound) {
		server.writeError(writer, request, http.StatusNotFound, "file_not_found", "File was not found.", nil)
		return
	}
	if errors.Is(err, sandbox.ErrNotRunning) {
		server.writeError(writer, request, http.StatusConflict, "state_conflict", "Sandbox is not running.", nil)
		return
	}
	var transition sandbox.InvalidStateTransition
	if errors.As(err, &transition) {
		status, code, message, details := serviceError(err)
		server.writeError(writer, request, status, code, message, details)
		return
	}
	server.writeError(writer, request, http.StatusConflict, "filesystem_conflict", "The runtime could not complete the file operation.", nil)
}

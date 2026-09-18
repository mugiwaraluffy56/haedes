package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/api"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/auth"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/fakes"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

const (
	defaultBind    = "0.0.0.0:8080"
	defaultRuntime = "http://sandbox-runtime:8080"
	defaultAPIKey  = "local-api-key"
	defaultOwner   = "local-owner"
)

func main() {
	clock := fakes.NewClock(time.Now().UTC())
	runtime := fakes.NewRuntime(clock)
	compute := fakes.NewCompute(clock)
	compute.StartTask = sandbox.TaskRef{
		ARN:      "local-fake-task",
		Endpoint: envOr("HAEDES_RUNTIME_ENDPOINT", defaultRuntime),
	}
	computeAdapter := &localCompute{compute: compute, runtime: runtime}

	service := sandbox.NewService(sandbox.Dependencies{
		Repository:    fakes.NewRepository(),
		Compute:       computeAdapter,
		Runtime:       runtime,
		Snapshots:     fakes.NewSnapshotRepository(),
		SnapshotStore: fakes.NewObjectStore(),
		IDs:           &localIDs{},
		Clock:         clock,
	})

	apiKey := envOr("HAEDES_LOCAL_API_KEY", defaultAPIKey)
	record, err := auth.NewAPIKeyRecord("local-key", defaultOwner, apiKey)
	if err != nil {
		log.Fatalf("create local API key: %v", err)
	}
	store, err := auth.NewMemoryStore(record)
	if err != nil {
		log.Fatalf("create local API key store: %v", err)
	}
	authentication, err := auth.NewService(store)
	if err != nil {
		log.Fatalf("create local authenticator: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", healthHandler)
	mux.HandleFunc("/readyz", readyHandler)
	mux.Handle("/v1/", api.NewServer(service, authentication))

	server := &http.Server{
		Addr:              envOr("HAEDES_CONTROL_PLANE_BIND", defaultBind),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownContext); err != nil {
			log.Printf("control plane shutdown: %v", err)
		}
	}()

	log.Printf("local control plane listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("control plane server: %v", err)
	}
}

func healthHandler(writer http.ResponseWriter, _ *http.Request) {
	writeStatus(writer, "ok")
}

func readyHandler(writer http.ResponseWriter, _ *http.Request) {
	writeStatus(writer, "ready")
}

func writeStatus(writer http.ResponseWriter, status string) {
	writer.Header().Set("Content-Type", "application/json")
	_, _ = writer.Write([]byte(fmt.Sprintf(`{"status":%q}`+"\n", status)))
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

type localCompute struct {
	compute *fakes.Compute
	runtime *fakes.Runtime
}

func (compute *localCompute) Start(ctx context.Context, request sandbox.ProvisionRequest) (sandbox.TaskRef, error) {
	task, err := compute.compute.Start(ctx, request)
	if err != nil {
		return sandbox.TaskRef{}, err
	}
	compute.runtime.Register(request.SandboxID, task)
	return task, nil
}

func (compute *localCompute) Stop(ctx context.Context, taskARN string) error {
	return compute.compute.Stop(ctx, taskARN)
}

func (compute *localCompute) Describe(ctx context.Context, taskARN string) (sandbox.TaskStatus, error) {
	return compute.compute.Describe(ctx, taskARN)
}

type localIDs struct {
	sandbox  atomic.Uint64
	command  atomic.Uint64
	snapshot atomic.Uint64
	token    atomic.Uint64
}

func (ids *localIDs) NewSandboxID() sandbox.SandboxID {
	return sandbox.SandboxID(fmt.Sprintf("sbx_local_%d", ids.sandbox.Add(1)))
}

func (ids *localIDs) NewCommandID() sandbox.CommandID {
	return sandbox.CommandID(fmt.Sprintf("cmd_local_%d", ids.command.Add(1)))
}

func (ids *localIDs) NewSnapshotID() sandbox.SnapshotID {
	return sandbox.SnapshotID(fmt.Sprintf("snp_local_%d", ids.snapshot.Add(1)))
}

func (ids *localIDs) NewRuntimeToken() string {
	return "local-runtime-token-" + strconv.FormatUint(ids.token.Add(1), 10)
}

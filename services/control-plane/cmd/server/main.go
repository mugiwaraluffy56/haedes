package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awsDynamo "github.com/aws/aws-sdk-go-v2/service/dynamodb"
	awsECS "github.com/aws/aws-sdk-go-v2/service/ecs"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/api"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/auth"
	internalaws "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/aws"
	awsDynamoAdapter "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/aws/dynamodb"
	awsECSAdapter "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/aws/ecs"
	awsS3Adapter "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/aws/s3"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/config"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/fakes"
	runtimeclient "github.com/mugiwaraluffy56/haedes/services/control-plane/internal/runtime"
	"github.com/mugiwaraluffy56/haedes/services/control-plane/internal/sandbox"
)

const (
	defaultBind    = "0.0.0.0:8080"
	defaultRuntime = "http://sandbox-runtime:8080"
	defaultAPIKey  = "local-api-key"
	defaultOwner   = "local-owner"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}
	service, err := buildService(settings)
	if err != nil {
		log.Fatalf("build control-plane service: %v", err)
	}

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
	apiServer := api.NewServer(service, authentication)
	mux.Handle("/v1/", apiServer)
	mux.Handle("/metrics", apiServer.MetricsHandler())

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

	log.Printf("control plane listening on %s (aws=%t)", server.Addr, settings.AWSEnabled)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("control plane server: %v", err)
	}
}

func buildService(settings config.Config) (*sandbox.Service, error) {
	if !settings.AWSEnabled {
		clock := fakes.NewClock(time.Now().UTC())
		runtime := fakes.NewRuntime(clock)
		compute := fakes.NewCompute(clock)
		compute.StartTask = sandbox.TaskRef{
			ARN: "local-fake-task", Endpoint: envOr("HAEDES_RUNTIME_ENDPOINT", defaultRuntime),
		}
		return sandbox.NewService(sandbox.Dependencies{
			Repository: fakes.NewRepository(), Compute: &localCompute{compute: compute, runtime: runtime}, Runtime: runtime,
			Snapshots: fakes.NewSnapshotRepository(), SnapshotStore: fakes.NewObjectStore(), IDs: &localIDs{}, Clock: clock,
		}), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	awsSettings, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(settings.AWSRegion))
	if err != nil {
		return nil, fmt.Errorf("load AWS SDK configuration: %w", err)
	}
	retryPolicy := internalaws.DefaultRetryPolicy()
	repository, err := awsDynamoAdapter.NewRepository(awsDynamo.NewFromConfig(awsSettings), awsDynamoAdapter.Config{
		TableName: settings.DynamoDBTable, IndexName: settings.DynamoDBIndex, Retry: retryPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("configure DynamoDB repository: %w", err)
	}
	uploader := manager.NewUploader(awsS3.NewFromConfig(awsSettings))
	snapshotStore, err := awsS3Adapter.NewStore(awsS3.NewFromConfig(awsSettings), uploader, awsS3Adapter.Config{Bucket: settings.SnapshotBucket, Retry: retryPolicy})
	if err != nil {
		return nil, fmt.Errorf("configure S3 snapshot store: %w", err)
	}
	provisioner, err := awsECSAdapter.NewProvisioner(awsECS.NewFromConfig(awsSettings), awsECSAdapter.Config{
		Cluster: settings.ECSCluster, TaskDefinition: settings.ECSTaskDefinition, ContainerName: settings.ECSContainerName,
		PrivateSubnetIDs: settings.PrivateSubnetIDs, SecurityGroupIDs: settings.SandboxSecurityGroupIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("configure ECS provisioner: %w", err)
	}
	snapshotRepository, err := awsDynamoAdapter.NewSnapshotRepository(awsDynamo.NewFromConfig(awsSettings), awsDynamoAdapter.Config{
		TableName: settings.SnapshotTable, IndexName: "sandbox-createdAt-index", Retry: retryPolicy,
	})
	if err != nil {
		return nil, fmt.Errorf("configure DynamoDB snapshot repository: %w", err)
	}
	clock := wallClock{}
	return sandbox.NewService(sandbox.Dependencies{
		Repository: repository, Compute: provisioner, Runtime: runtimeclient.NewClient(nil),
		Snapshots: snapshotRepository, SnapshotStore: snapshotStore, IDs: &productionIDs{}, Clock: clock,
	}), nil
}

type wallClock struct{}

func (wallClock) Now() time.Time { return time.Now().UTC() }

type productionIDs struct{ counter atomic.Uint64 }

func (ids *productionIDs) next(prefix string) string {
	return fmt.Sprintf("%s%x%x", prefix, time.Now().UnixNano(), ids.counter.Add(1))
}

func (ids *productionIDs) NewSandboxID() sandbox.SandboxID {
	return sandbox.SandboxID(ids.next("sbx_"))
}
func (ids *productionIDs) NewCommandID() sandbox.CommandID {
	return sandbox.CommandID(ids.next("cmd_"))
}
func (ids *productionIDs) NewSnapshotID() sandbox.SnapshotID {
	return sandbox.SnapshotID(ids.next("snp_"))
}
func (ids *productionIDs) NewRuntimeToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err == nil {
		return "runtime-" + hex.EncodeToString(bytes)
	}
	return "runtime-" + ids.next("token_")
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

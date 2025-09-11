package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"src/api"
	"src/config"
	"src/kafka"
	"src/logger"
	"src/metrics"
	"src/models"
	oidcutil "src/oidc"
	"src/store"
	"syscall"
	"time"

	skafka "github.com/segmentio/kafka-go"
)

var broadcast = make(chan models.Message)
var appCtx context.Context
var appCancel context.CancelFunc

func main() {
	logger.Info("starting application")

	// Root context with cancellation for graceful shutdown (used across subsystems)
	appCtx, appCancel = context.WithCancel(context.Background())
	defer appCancel()

	// Initialize Mongo store layer
	if err := store.Init(appCtx); err != nil {
		log.Fatalf("mongo init failed: %v", err)
	}
	defer store.Close(context.Background())

	// Initialize OIDC (provider + verifier)
	_, coreVerifier := oidcutil.Init(appCtx)
	verifier := &oidcutil.Verifier{Fn: func(ctx context.Context, raw string) error {
		_, err := oidcutil.VerifyToken(ctx, coreVerifier, raw)
		return err
	}}

	// Capture OS signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		logger.Info("shutdown signal received")
		appCancel()
		// Allow a short drain period
		time.Sleep(500 * time.Millisecond)
		os.Exit(0)
	}()

	// Start Kafka reader consumer -> broadcast channel
	go kafka.Reader(appCtx, broadcast)

	maxLen := api.ParseMaxLen(os.Getenv("MESSAGE_MAX_LENGTH"), 1000)
	validator := api.NewMessageValidator("../schema.json")
	producer := kafka.ProducerAdapter{}
	repo := store.RepositoryAdapter{}
	server := api.NewServer(producer, repo, verifier, validator, broadcast, maxLen)

	http.HandleFunc("/healthz", handleHealth)
	http.HandleFunc("/readyz", handleReady)
	http.HandleFunc("/metrics", metrics.Handler)
	http.Handle("/", server) // server handles its subpaths

	logger.Info("http server listening", logger.FieldKV("port", config.ApiPort))
	if err := http.ListenAndServe(":"+config.ApiPort, nil); err != nil {
		logger.Error("http server error", err)
	}
}

// keep health/ready/metrics handlers below

// Health endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// Readiness endpoint
func handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 1*time.Second)
	defer cancel()
	if err := store.Ping(ctx); err != nil {
		http.Error(w, "mongo not ready", http.StatusServiceUnavailable)
		return
	}
	// Kafka readiness: attempt to create a reader and fetch 0 messages via offset check
	kr := skafka.NewReader(skafka.ReaderConfig{Brokers: []string{config.KafkaBroker}, Topic: config.Topic, Partition: 0, MinBytes: 1, MaxBytes: 1})
	defer kr.Close()
	// Use context timeout; try a single fetch of offset to validate metadata
	_, err := kr.ReadLag(ctx)
	if err != nil {
		http.Error(w, "kafka not ready", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ready"))
}

// ensureMongoIndexes creates required indexes (idempotent). Unique index on message_id and TTL/index on timestamp for sorting.
// (Mongo index creation moved to store package)

// (OIDC provider/backoff logic moved to oidc/oidc.go)

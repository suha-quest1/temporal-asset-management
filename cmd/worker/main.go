package main

import (
	"context"
	"log"
	"os"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/vishworks/assetmgmt/internal/activities"
	appdb "github.com/vishworks/assetmgmt/internal/db"
	"github.com/vishworks/assetmgmt/internal/workflows"
)

func main() {
	temporalAddr := envOrDefault("TEMPORAL_ADDRESS", "localhost:7233")
	databaseURL := envOrDefault("DATABASE_URL", "postgres://assetmgmt:assetmgmt@localhost:5432/assetmgmt?sslmode=disable")
	sesURL := envOrDefault("SES_URL", "http://localhost:8081")
	mlScorerURL := envOrDefault("ML_SCORER_URL", "http://localhost:5001")
	creditFacURL := envOrDefault("CREDIT_FACILITY_URL", "http://localhost:8082")
	reportGenURL := envOrDefault("REPORT_GENERATOR_URL", "http://localhost:5002")

	ctx := context.Background()

	// Connect to Postgres
	pool, err := appdb.Connect(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Create Temporal client
	c, err := client.Dial(client.Options{HostPort: temporalAddr})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, workflows.TaskQueue, worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.CapitalCallWorkflow)
	w.RegisterWorkflow(workflows.LPResponseWorkflow)

	// Register activities
	acts := activities.NewActivities(pool, sesURL, mlScorerURL, creditFacURL, reportGenURL)
	w.RegisterActivity(acts)

	log.Println("Worker starting on task queue:", workflows.TaskQueue)
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalf("Worker failed: %v", err)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"miniot/apps/ingestor/internal/config"
	"miniot/apps/ingestor/internal/health"
	"miniot/apps/ingestor/internal/mqtt"
	"miniot/apps/ingestor/internal/processor"
	"miniot/apps/ingestor/internal/storage"
)

func main() {
	cfg := config.LoadConfig()
	log.Printf("[minIoT Ingestor] Initializing telemetry ingestor service...")

	store, err := storage.NewStorage(cfg)
	if err != nil {
		log.Fatalf("Storage initialization failed: %v", err)
	}
	defer store.Pool.Close()

	proc, err := processor.NewTelemetryProcessor(store)
	if err != nil {
		log.Fatalf("Processor initialization failed: %v", err)
	}
	subscriber := mqtt.NewSubscriber(cfg, proc)

	if err := subscriber.Start(); err != nil {
		log.Fatalf("MQTT subscriber start failed: %v", err)
	}
	defer subscriber.Stop()

	healthServer := health.NewServer(cfg.HealthPort, store, subscriber)
	_ = healthServer.Start()
	log.Printf("[minIoT Ingestor] Health probe server listening on :%s/health", cfg.HealthPort)

	log.Printf("[minIoT Ingestor] Ready and listening for device telemetry on %s", cfg.MQTTTopic)
	waitForShutdown()
	_ = healthServer.Stop(context.Background())
}

func waitForShutdown() {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)
	<-stopChan
	log.Println("[minIoT Ingestor] Shutting down gracefully...")
}

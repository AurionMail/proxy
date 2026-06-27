package main

import (
	"log"

	"aurion/proxy/internal/config"
	"aurion/proxy/internal/logging"
	"aurion/proxy/internal/queue"
	"aurion/proxy/internal/smtpserver"
)

func main() {
	// Load .env + defaults
	cfg := config.Load()

	// Structured logging
	logging.Init()

	// Init queue + forwarder
	queue.InitQueue(cfg.QueueSize, cfg.ForwardAddr)
	queue.StartWorkers(cfg.WorkerCount)

	// Start SMTP proxy
	log.Printf("Starting Aurion SMTP Proxy on %s", cfg.ListenAddr)
	smtpserver.Start(cfg)
}

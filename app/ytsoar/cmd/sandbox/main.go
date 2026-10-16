// The sandbox hosts ALL dynamic code - Python connectors, JS connectors and
// code snippets - behind the ConnectorRuntime gRPC API. Every run is a fresh
// subprocess (localexec). It deliberately has no database or message-queue
// access: deploy it with zero credentials in its environment.
package main

import (
	"log"

	"github.com/yuudev14/ytsoar/internal/app"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func main() {
	cfg := config.Load()
	appLogger := logger.SetupLogger()
	defer appLogger.Sync()

	server, err := app.NewSandbox(cfg, appLogger)
	if err != nil {
		log.Fatal(err)
	}

	appLogger.Infow("sandbox started",
		"listen", cfg.SandboxListenAddr,
		"connectors_dir", cfg.ConnectorsDir,
	)
	if err := server.Serve(cfg.SandboxListenAddr); err != nil {
		log.Fatalf("sandbox server stopped: %v", err)
	}
}

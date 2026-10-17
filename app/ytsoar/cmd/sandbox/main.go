// The sandbox runs ALL dynamic code (Python connectors, JS connectors, code snippets)
// behind the ConnectorRuntime gRPC API. Each run is a new subprocess (localexec). It has
// no database or message queue access on purpose: deploy it with no credentials in its
// environment.
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
	defer func() { _ = appLogger.Sync() }()

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

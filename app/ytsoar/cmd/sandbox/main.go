// The sandbox hosts ALL dynamic code — Python connectors, JS connectors and
// code snippets — behind the ConnectorRuntime gRPC API. Every run is a fresh
// subprocess (localexec). It deliberately has no database or message-queue
// access: deploy it with zero credentials in its environment.
package main

import (
	"log"
	"time"

	"github.com/yuudev14/ytsoar/internal/adapters/grpcserver"
	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/localexec"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logger"
)

const defaultNodeTimeout = 5 * time.Minute

func main() {
	cfg := config.Load()
	appLogger := logger.SetupLogger()
	defer appLogger.Sync()

	// Code nodes are pinned first; connector ids from the unified tree never
	// override them.
	byConnector := map[string]execution.NodeRuntime{
		"code_snippet":    localexec.NewPythonRunner(appLogger),
		"code_snippet_js": localexec.NewNodeRunner(appLogger),
	}

	// Python connectors are the default: any connector without an explicit
	// mapping runs through the python connector harness. Connectors whose
	// info.json declares "runtime": "node" run through the node harness.
	var defaultRuntime execution.NodeRuntime
	if cfg.ConnectorsDir != "" {
		pythonConnectorRunner, err := localexec.NewPythonConnectorRunner(appLogger, cfg.ConnectorsDir)
		if err != nil {
			log.Fatalf("failed to setup python connector runner: %v", err)
		}
		defaultRuntime = pythonConnectorRunner

		jsConnectorRunner, err := localexec.NewNodeConnectorRunner(appLogger, cfg.ConnectorsDir)
		if err != nil {
			log.Fatalf("failed to setup js connector runner: %v", err)
		}
		jsConnectors, err := localexec.ListNodeConnectors(cfg.ConnectorsDir)
		if err != nil {
			appLogger.Warnf("could not scan connectors tree %s: %v", cfg.ConnectorsDir, err)
		}
		for _, id := range jsConnectors {
			if _, pinned := byConnector[id]; !pinned {
				byConnector[id] = jsConnectorRunner
			}
		}
		appLogger.Infow("registered js connectors", "connectors", jsConnectors)
	} else {
		appLogger.Warn("CONNECTORS_DIR is empty: connectors are disabled, only code nodes will run")
	}

	resolver := execution.NewStaticResolver(defaultRuntime, byConnector)
	server := grpcserver.NewRuntimeServer(appLogger, resolver, defaultNodeTimeout)

	appLogger.Infow("sandbox started",
		"listen", cfg.SandboxListenAddr,
		"connectors_dir", cfg.ConnectorsDir,
	)
	if err := server.Serve(cfg.SandboxListenAddr); err != nil {
		log.Fatalf("sandbox server stopped: %v", err)
	}
}

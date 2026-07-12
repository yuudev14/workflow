// The sandbox hosts ALL dynamic code — Python connectors, JS connectors and
// code snippets — behind the ConnectorRuntime gRPC API. Every run is a fresh
// subprocess (localexec). It deliberately has no database or message-queue
// access: deploy it with zero credentials in its environment.
package main

import (
	"fmt"
	"log"

	"github.com/yuudev14/ytsoar/internal/adapters/grpcserver"
	"github.com/yuudev14/ytsoar/internal/adapters/runtimes/localexec"
	"github.com/yuudev14/ytsoar/internal/application/execution"
	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/domain"
	"github.com/yuudev14/ytsoar/internal/logger"
)

func main() {
	cfg := config.Load()
	appLogger := logger.SetupLogger()
	defer appLogger.Sync()

	// Code nodes are pinned first; connector ids from the unified tree never
	// override them.
	byConnector := map[string]execution.NodeRuntime{
		"code_snippet_py": localexec.NewPythonRunner(appLogger, cfg.PythonMemoryLimitMB),
		"code_snippet_js": localexec.NewNodeRunner(appLogger, cfg.NodeMemoryLimitMB),
	}

	// Python connectors are the default: any connector without an explicit
	// mapping runs through the python connector harness. Connectors whose
	// info.json declares "runtime": "node" run through the node harness —
	// checked per request, so connectors uploaded after boot route correctly
	// without a restart.
	var defaultRuntime execution.NodeRuntime
	var jsConnectorRunner execution.NodeRuntime
	if cfg.ConnectorsDir != "" {
		pythonConnectorRunner, err := localexec.NewPythonConnectorRunner(appLogger, cfg.ConnectorsDir, cfg.PythonMemoryLimitMB)
		if err != nil {
			log.Fatalf("failed to setup python connector runner: %v", err)
		}
		defaultRuntime = pythonConnectorRunner

		nodeConnectorRunner, err := localexec.NewNodeConnectorRunner(appLogger, cfg.ConnectorsDir, cfg.NodeMemoryLimitMB)
		if err != nil {
			log.Fatalf("failed to setup js connector runner: %v", err)
		}
		jsConnectorRunner = nodeConnectorRunner
	} else {
		appLogger.Warn("CONNECTORS_DIR is empty: connectors are disabled, only code nodes will run")
	}

	resolver := execution.RuntimeResolverFunc(func(task domain.Tasks) (execution.NodeRuntime, error) {
		if task.ConnectorID != nil {
			if runtime, ok := byConnector[*task.ConnectorID]; ok {
				return runtime, nil
			}
			if jsConnectorRunner != nil && localexec.IsNodeConnector(cfg.ConnectorsDir, *task.ConnectorID) {
				return jsConnectorRunner, nil
			}
		}
		if defaultRuntime == nil {
			return nil, fmt.Errorf("no runtime registered for connector %v", task.ConnectorID)
		}
		return defaultRuntime, nil
	})
	server := grpcserver.NewRuntimeServer(appLogger, resolver, cfg.NodeTimeout)

	appLogger.Infow("sandbox started",
		"listen", cfg.SandboxListenAddr,
		"connectors_dir", cfg.ConnectorsDir,
	)
	if err := server.Serve(cfg.SandboxListenAddr); err != nil {
		log.Fatalf("sandbox server stopped: %v", err)
	}
}

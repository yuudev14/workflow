// The sandbox hosts ALL dynamic code - Python connectors, JS connectors and
// code snippets - behind the ConnectorRuntime gRPC API. Every run is a fresh
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
	byConnector := map[string]execution.TaskRuntime{
		"code_snippet_py": localexec.NewPythonSnippetRuntime(appLogger, cfg.PythonMemoryLimitMB),
		"code_snippet_js": localexec.NewNodeJSSnippetRuntime(appLogger, cfg.NodeMemoryLimitMB),
	}

	// Python connectors are the default: any connector without an explicit
	// mapping runs through the python connector harness. Connectors whose
	// info.json declares "runtime": "node" run through the node harness -
	// checked per request, so connectors uploaded after boot route correctly
	// without a restart.
	var defaultRuntime execution.TaskRuntime
	var nodeJSConnectorRuntime execution.TaskRuntime
	if cfg.ConnectorsDir != "" {
		pythonConnectorRuntime, err := localexec.NewPythonConnectorRuntime(appLogger, cfg.ConnectorsDir, cfg.PythonMemoryLimitMB)
		if err != nil {
			log.Fatalf("failed to setup python connector runtime: %v", err)
		}
		defaultRuntime = pythonConnectorRuntime

		nodeJSRuntime, err := localexec.NewNodeJSConnectorRuntime(appLogger, cfg.ConnectorsDir, cfg.NodeMemoryLimitMB)
		if err != nil {
			log.Fatalf("failed to setup js connector runtime: %v", err)
		}
		nodeJSConnectorRuntime = nodeJSRuntime
	} else {
		appLogger.Warn("CONNECTORS_DIR is empty: connectors are disabled, only code nodes will run")
	}

	// DynamicResolver runs in the sandbox. It picks the runtime for each task that
	// the worker sends over gRPC, and it reads info.json on every call, so a
	// connector that you upload works with no restart.
	resolver := execution.DynamicResolver(func(task domain.Tasks) (execution.TaskRuntime, error) {
		if task.ConnectorID != nil {
			if runtime, ok := byConnector[*task.ConnectorID]; ok {
				return runtime, nil
			}
			if nodeJSConnectorRuntime != nil && localexec.IsNodeJSConnector(cfg.ConnectorsDir, *task.ConnectorID) {
				return nodeJSConnectorRuntime, nil
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

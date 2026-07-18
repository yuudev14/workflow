// Package depsinstall vendors a connector's declared dependencies right
// after upload — the same commands `make connector-deps` runs for the whole
// tree. It executes in the API container, whose alpine/musl toolchain
// matches the sandbox image that will import the packages.
package depsinstall

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/yuudev14/ytsoar/internal/logger"
)

const installTimeout = 5 * time.Minute

// LocalInstaller implements connectors.DepsInstaller.
type LocalInstaller struct {
	logger logger.Logger
	dir    string
}

func New(log logger.Logger, dir string) *LocalInstaller {
	return &LocalInstaller{
		logger: log,
		dir:    dir,
	}
}

// Install vendors requirements.txt into <id>/deps and package.json into
// <id>/node_modules. Connectors without dependency files are a no-op.
func (i *LocalInstaller) Install(ctx context.Context, connectorID string) error {
	base := filepath.Join(i.dir, connectorID)

	if _, err := os.Stat(filepath.Join(base, "requirements.txt")); err == nil {
		i.logger.Infow("installing python deps", "connector", connectorID)
		if err := i.run(ctx, base, "pip", "install", "--break-system-packages",
			"--no-cache-dir", "--quiet", "--target", "deps", "-r", "requirements.txt"); err != nil {
			return err
		}
	}
	if _, err := os.Stat(filepath.Join(base, "package.json")); err == nil {
		i.logger.Infow("installing node deps", "connector", connectorID)
		if err := i.run(ctx, base, "npm", "install", "--ignore-scripts",
			"--omit=dev", "--no-audit", "--no-fund"); err != nil {
			return err
		}
	}
	return nil
}

func (i *LocalInstaller) run(ctx context.Context, dir string, name string, args ...string) error {
	runCtx, cancel := context.WithTimeout(ctx, installTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %v: %w\n%s", name, args, err, out)
	}
	return nil
}

package repository_test

import (
	"os"
	"testing"

	"github.com/yuudev14-workflow/workflow-service/environment"
	"github.com/yuudev14-workflow/workflow-service/internal/logging"
)

func TestMain(m *testing.M) {
	environment.Setup()
	logging.Setup("DEBUG")
	os.Exit(m.Run())
}

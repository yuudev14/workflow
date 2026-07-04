package repository

import (
	"os"
	"testing"

	"github.com/yuudev14/ytsoar/internal/config"
	"github.com/yuudev14/ytsoar/internal/logging"
)

func TestMain(m *testing.M) {
	config.Load()
	logging.Setup("DEBUG")
	os.Exit(m.Run())
}

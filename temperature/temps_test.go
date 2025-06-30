package temperature

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
)

func TestTemperature(t *testing.T) {
	logger := zerolog.New(os.Stdout).Level(zerolog.DebugLevel)
	temperature := New(&logger)
	temperature.Fetch()
}

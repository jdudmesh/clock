package sunset

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestSunset(t *testing.T) {
	assert := assert.New(t)
	logger := zerolog.New(os.Stdout).Level(zerolog.DebugLevel)

	sunset := New(&logger)
	assert.NotNil(sunset)

	sunset.Fetch()
	assert.NotNil(sunset.data)

	sunriseTime, sunsetTime := sunset.Get()
	assert.NotNil(sunriseTime)
	assert.NotNil(sunsetTime)

	assert.True(sunriseTime.Before(sunsetTime))
}

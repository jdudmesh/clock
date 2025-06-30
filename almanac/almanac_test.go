package almanac

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestAlmanac(t *testing.T) {
	assert := assert.New(t)
	logger := zerolog.New(os.Stdout).Level(zerolog.DebugLevel)

	almanac := New(&logger)
	assert.NotNil(almanac)

	almanac.Fetch()
	assert.NotNil(almanac.data)

	sunriseTime := almanac.GetSunrise()
	sunsetTime := almanac.GetSunset()
	assert.NotNil(sunriseTime)
	assert.NotNil(sunsetTime)

	assert.True(sunriseTime.Before(sunsetTime))
}

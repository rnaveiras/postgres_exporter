package collector

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlatformFrom(t *testing.T) {
	t.Parallel()

	assert.Equal(t, PlatformCommunity, platformFrom(false, false, false, false))
	assert.Equal(t, PlatformAurora, platformFrom(true, true, false, false), "aurora also has rds_superuser")
	assert.Equal(t, PlatformRDS, platformFrom(false, true, false, false))
	assert.Equal(t, PlatformCloudSQL, platformFrom(false, false, true, false))
	assert.Equal(t, PlatformAzureFlexible, platformFrom(false, false, false, true))
}

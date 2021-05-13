package version

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersion(t *testing.T) {
	zeroVer, err := Parse("")
	require.NoError(t, err)
	// Equal
	assert.Zero(t, zeroVer.Cmp(big.NewInt(0)))

	ver, err := Parse("1")
	require.NoError(t, err)
	assert.Zero(t, ver.Cmp(big.NewInt(1)))
	newVer, err := IncreaseVersion(ver.String())
	require.NoError(t, err)
	assert.Equal(t, "2", newVer)
}

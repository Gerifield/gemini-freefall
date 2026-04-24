package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	c, err := LoadConfig("../../config.yaml.example", false)
	require.NoError(t, err)

	require.Len(t, c.Backend, 2)
	assert.Equal(t, "backend1", c.Backend[0].Name)
	require.Len(t, c.Backend[0].Models, 3)
	assert.Equal(t, "gemini-3.1-pro", c.Backend[0].Models[0])

	assert.True(t, isValidBackend("backend1.gemini-3.1-flash", c))
	assert.True(t, isValidBackend("backend2.gemini-3.1-flash", c))
	assert.False(t, isValidBackend("backend1.gemini-3.1-nope", c))
	assert.False(t, isValidBackend("backend66.gemini-3.1-flash", c))

	testB, err := getBackend("backend2.gemini-3.1-flash", c)
	require.NoError(t, err)
	assert.Equal(t, "<YOUR API KEY HERE2>", testB.Key)
}

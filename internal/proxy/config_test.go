package proxy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	t.Run("Default Gemini Mode (from example)", func(t *testing.T) {
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
	})

	t.Run("OpenAI Mode Filters Gemini", func(t *testing.T) {
		c, err := LoadConfig("config_mixed_test.yaml", true)
		require.NoError(t, err)

		// OpenAI mode should drop the gemini-native backend
		require.Len(t, c.Backend, 3)
		assert.Equal(t, "openai-backend", c.Backend[0].Name)
		assert.Equal(t, "anthropic-backend", c.Backend[1].Name)
		assert.Equal(t, "custom-local", c.Backend[2].Name)

		// The proxy route should only contain non-gemini targets
		require.Len(t, c.Config.Proxy["mixed-path"], 3)
		assert.Equal(t, "openai-backend.gpt-4", c.Config.Proxy["mixed-path"][0])
	})

	t.Run("Gemini Mode Filters OpenAI", func(t *testing.T) {
		c, err := LoadConfig("config_mixed_test.yaml", false)
		require.NoError(t, err)

		// Gemini mode should drop the non-gemini backends
		require.Len(t, c.Backend, 1)
		assert.Equal(t, "gemini-native", c.Backend[0].Name)

		// The proxy route should only contain gemini targets
		require.Len(t, c.Config.Proxy["mixed-path"], 1)
		assert.Equal(t, "gemini-native.gemini-pro", c.Config.Proxy["mixed-path"][0])
	})
}

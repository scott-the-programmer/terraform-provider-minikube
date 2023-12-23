package state_utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryConverter(t *testing.T) {
	converter := MemoryConverter()

	t.Run("ValidMemoryFlagG", func(t *testing.T) {
		result := converter("2G")
		assert.Equal(t, "2048mb", result)
	})

	t.Run("ValidMemoryFlagGb", func(t *testing.T) {
		result := converter("2Gb")
		assert.Equal(t, "2048mb", result)
	})

	t.Run("Equivalence", func(t *testing.T) {
		resultg := converter("2Gb")
		resultm := converter("2048mb")
		assert.Equal(t, resultm, resultg)
	})

	t.Run("InvalidMemoryFlag", func(t *testing.T) {
		assert.PanicsWithError(t, "memory flag is not a string", func() {
			converter(123)
		})
	})

	t.Run("InvalidMemoryValue", func(t *testing.T) {
		assert.PanicsWithError(t, "invalid memory value", func() {
			converter("invalid")
		})
	})
}

func TestMemoryValidator(t *testing.T) {
	validator := MemoryValidator()

	t.Run("ValidMemoryFlag", func(t *testing.T) {
		diags := validator("2G", nil)
		assert.Empty(t, diags)
	})

	t.Run("InvalidMemoryFlag", func(t *testing.T) {
		diags := validator(123, nil)
		require.Len(t, diags, 1)
		assert.Equal(t, diags[0].Summary, "memory flag is not a string")
	})

	t.Run("InvalidMemoryValue", func(t *testing.T) {
		diags := validator("invalid", nil)
		require.Len(t, diags, 1)
		assert.True(t, diags.HasError())
	})

}

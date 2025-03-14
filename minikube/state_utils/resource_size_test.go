package state_utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResourceSizeConverter(t *testing.T) {
	converter := ResourceSizeConverter()

	t.Run("ValidSizeFlagG", func(t *testing.T) {
		result := converter("2G")
		assert.Equal(t, "2048mb", result)
	})

	t.Run("ValidSizeFlagGb", func(t *testing.T) {
		result := converter("2Gb")
		assert.Equal(t, "2048mb", result)
	})

	t.Run("Equivalence", func(t *testing.T) {
		resultg := converter("2Gb")
		resultm := converter("2048mb")
		assert.Equal(t, resultm, resultg)
	})

	t.Run("InvalidSizeFlag", func(t *testing.T) {
		assert.PanicsWithError(t, "resource size is not a string", func() {
			converter(123)
		})
	})

	t.Run("InvalidSizeValue", func(t *testing.T) {
		assert.PanicsWithError(t, "invalid resource size value", func() {
			converter("invalid")
		})
	})
}

func TestResourceSizeValidator(t *testing.T) {
	validator := ResourceSizeValidator()

	t.Run("ValidSizeFlag", func(t *testing.T) {
		diags := validator("2G", nil)
		assert.Empty(t, diags)
	})

	t.Run("InvalidSizeFlag", func(t *testing.T) {
		diags := validator(123, nil)
		require.Len(t, diags, 1)
		assert.Equal(t, diags[0].Summary, "resource size is not a string")
	})

	t.Run("InvalidSizeValue", func(t *testing.T) {
		diags := validator("invalid", nil)
		require.Len(t, diags, 1)
		assert.True(t, diags.HasError())
	})
}

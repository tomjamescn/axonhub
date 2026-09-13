package objects

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFieldMaps(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		err := ValidateFieldMaps([]RequestRewriteFieldMap{
			{Path: "reasoning_effort", Values: []RequestRewriteValueMap{{From: "high", To: "xhigh"}}},
		})
		require.NoError(t, err)
	})

	t.Run("empty path", func(t *testing.T) {
		err := ValidateFieldMaps([]RequestRewriteFieldMap{
			{Path: "", Values: []RequestRewriteValueMap{{From: "a", To: "b"}}},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyRequestRewriteField)
	})

	t.Run("no values", func(t *testing.T) {
		err := ValidateFieldMaps([]RequestRewriteFieldMap{
			{Path: "reasoning_effort"},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrEmptyRequestRewriteField)
	})

	t.Run("duplicate path", func(t *testing.T) {
		err := ValidateFieldMaps([]RequestRewriteFieldMap{
			{Path: "temperature", Values: []RequestRewriteValueMap{{From: "1", To: "2"}}},
			{Path: "temperature", Values: []RequestRewriteValueMap{{From: "3", To: "4"}}},
		})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrDuplicateRequestRewriteField)
	})
}

func TestApplyFieldMapsTo(t *testing.T) {
	fields := []RequestRewriteFieldMap{
		{Path: "reasoning_effort", Values: []RequestRewriteValueMap{{From: "high", To: "xhigh"}, {From: "", To: "low"}}},
	}

	got, ok := ApplyFieldMapsTo(fields, "reasoning_effort", "high")
	assert.True(t, ok)
	assert.Equal(t, "xhigh", got)

	got, ok = ApplyFieldMapsTo(fields, "reasoning_effort", "")
	assert.True(t, ok)
	assert.Equal(t, "low", got)

	got, ok = ApplyFieldMapsTo(fields, "reasoning_effort", "medium")
	assert.False(t, ok)
	assert.Equal(t, "medium", got)

	got, ok = ApplyFieldMapsTo(fields, "temperature", "high")
	assert.False(t, ok)
	assert.Equal(t, "high", got)
}

package biz

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/objects"
)

func TestMatchedFields(t *testing.T) {
	rules := []*ent.RequestRewriteRule{
		{
			ID:           1,
			ModelPattern: "gpt-4o",
			FieldMaps: []objects.RequestRewriteFieldMap{
				{Path: "reasoning_effort", Values: []objects.RequestRewriteValueMap{{From: "high", To: "max"}}},
			},
		},
		{
			ID:           2,
			ModelPattern: "claude-.*",
			FieldMaps: []objects.RequestRewriteFieldMap{
				{Path: "temperature", Values: []objects.RequestRewriteValueMap{{From: "0.9", To: "0.7"}}},
			},
		},
	}

	t.Run("exact match", func(t *testing.T) {
		got := MatchedFields(rules, "gpt-4o")
		require.Len(t, got, 1)
		assert.Equal(t, "reasoning_effort", got[0].Path)
		assert.Equal(t, "max", got[0].Values[0].To)
	})

	t.Run("regex match", func(t *testing.T) {
		got := MatchedFields(rules, "claude-3-opus")
		require.Len(t, got, 1)
		assert.Equal(t, "temperature", got[0].Path)
	})

	t.Run("no match", func(t *testing.T) {
		got := MatchedFields(rules, "gemini-pro")
		assert.Empty(t, got)
	})

	t.Run("first matching rule wins for the same field and value", func(t *testing.T) {
		rules := []*ent.RequestRewriteRule{
			{
				ID:           1,
				ModelPattern: "*",
				FieldMaps: []objects.RequestRewriteFieldMap{
					{Path: "reasoning_effort", Values: []objects.RequestRewriteValueMap{{From: "high", To: "max"}}},
				},
			},
			{
				ID:           2,
				ModelPattern: "*",
				FieldMaps: []objects.RequestRewriteFieldMap{
					{Path: "reasoning_effort", Values: []objects.RequestRewriteValueMap{{From: "high", To: "xhigh"}}},
				},
			},
		}

		got := MatchedFields(rules, "gpt-4o")
		require.Len(t, got, 2)

		value, matched := objects.ApplyFieldMapsTo(got, "reasoning_effort", "high")
		assert.True(t, matched)
		assert.Equal(t, "max", value)
	})
}

package orchestrator

import (
	"context"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/server/biz"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/httpclient"
)

func runRequestRewrite(t *testing.T, rules []*ent.RequestRewriteRule, request *llm.Request) *llm.Request {
	t.Helper()

	provider := func(ctx context.Context) []*ent.RequestRewriteRule { return rules }
	result, err := applyRequestRewrite(provider).OnInboundLlmRequest(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)

	return result
}

func TestApplyRequestRewriteNilService(t *testing.T) {
	// A nil provider must produce a no-op middleware.
	request := &llm.Request{Model: "gpt-4o", ReasoningEffort: "high"}

	result, err := applyRequestRewrite(nil).OnInboundLlmRequest(context.Background(), request)
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "high", result.ReasoningEffort)

	// A nil concrete biz service must also yield a nil provider.
	provider := requestRewriteProviderFromService((*biz.RequestRewriteRuleService)(nil))
	assert.Nil(t, provider)
}

func TestApplyRequestRewrite(t *testing.T) {
	t.Run("nil service is a no-op", func(t *testing.T) {
		request := &llm.Request{Model: "gpt-4o", ReasoningEffort: "high"}
		result := runRequestRewrite(t, []*ent.RequestRewriteRule{}, request)
		assert.Equal(t, "high", result.ReasoningEffort)
	})

	t.Run("no rules is a no-op", func(t *testing.T) {
		request := &llm.Request{Model: "gpt-4o", ReasoningEffort: "high"}
		result := runRequestRewrite(t, nil, request)
		assert.Equal(t, "high", result.ReasoningEffort)
	})

	t.Run("rewrites reasoning_effort", func(t *testing.T) {
		rules := []*ent.RequestRewriteRule{
			{
				ID:           1,
				ModelPattern: "gpt-4o",
				FieldMaps: []objects.RequestRewriteFieldMap{
					{Path: "reasoning_effort", Values: []objects.RequestRewriteValueMap{{From: "high", To: "xhigh"}}},
				},
			},
		}

		request := &llm.Request{Model: "gpt-4o", ReasoningEffort: "high"}
		result := runRequestRewrite(t, rules, request)
		assert.Equal(t, "xhigh", result.ReasoningEffort)
	})

	t.Run("model pattern does not match", func(t *testing.T) {
		rules := []*ent.RequestRewriteRule{
			{
				ID:           1,
				ModelPattern: "gpt-4o",
				FieldMaps: []objects.RequestRewriteFieldMap{
					{Path: "reasoning_effort", Values: []objects.RequestRewriteValueMap{{From: "high", To: "xhigh"}}},
				},
			},
		}

		request := &llm.Request{Model: "claude-3-opus", ReasoningEffort: "high"}
		result := runRequestRewrite(t, rules, request)
		assert.Equal(t, "high", result.ReasoningEffort)
	})

	t.Run("matches the client original model, not a mapped one", func(t *testing.T) {
		// The middleware matches the model name as the client sent it, before
		// any API key profile model mapping.
		rules := []*ent.RequestRewriteRule{
			{
				ID:           1,
				ModelPattern: "client-model",
				FieldMaps: []objects.RequestRewriteFieldMap{
					{Path: "reasoning_effort", Values: []objects.RequestRewriteValueMap{{From: "high", To: "xhigh"}}},
				},
			},
		}

		request := &llm.Request{Model: "client-model", ReasoningEffort: "high"}
		result := runRequestRewrite(t, rules, request)
		assert.Equal(t, "xhigh", result.ReasoningEffort)
	})

	t.Run("empty value clears reasoning_effort and raw body", func(t *testing.T) {
		rules := []*ent.RequestRewriteRule{
			{
				ID:           1,
				ModelPattern: "gpt-4o",
				FieldMaps: []objects.RequestRewriteFieldMap{
					{Path: "reasoning_effort", Values: []objects.RequestRewriteValueMap{{From: "high", To: ""}}},
				},
			},
		}

		request := &llm.Request{
			Model:           "gpt-4o",
			ReasoningEffort: "high",
			RawRequest:      &httpclient.Request{Body: []byte(`{"model":"gpt-4o","reasoning_effort":"high"}`)},
		}
		result := runRequestRewrite(t, rules, request)
		assert.Equal(t, "", result.ReasoningEffort)
		assert.JSONEq(t, `{"model":"gpt-4o"}`, string(result.RawRequest.Body))
	})

	t.Run("rewrites numeric and pointer fields", func(t *testing.T) {
		rules := []*ent.RequestRewriteRule{
			{
				ID:           1,
				ModelPattern: "*",
				FieldMaps: []objects.RequestRewriteFieldMap{
					{Path: "temperature", Values: []objects.RequestRewriteValueMap{{From: "0.9", To: "0.5"}}},
					{Path: "max_tokens", Values: []objects.RequestRewriteValueMap{{From: "100000", To: "8192"}}},
				},
			},
		}

		request := &llm.Request{
			Model:       "gpt-4o",
			TopP:        lo.ToPtr(0.9),
			Temperature: lo.ToPtr(0.9),
			MaxTokens:   lo.ToPtr(int64(100000)),
		}
		result := runRequestRewrite(t, rules, request)
		require.NotNil(t, result.Temperature)
		assert.InDelta(t, 0.5, *result.Temperature, 1e-9)
		require.NotNil(t, result.MaxTokens)
		assert.Equal(t, int64(8192), *result.MaxTokens)
	})

	t.Run("unlisted field falls back to raw body", func(t *testing.T) {
		rules := []*ent.RequestRewriteRule{
			{
				ID:           1,
				ModelPattern: "*",
				FieldMaps: []objects.RequestRewriteFieldMap{
					{Path: "service_tier", Values: []objects.RequestRewriteValueMap{{From: "flex", To: "default"}}},
				},
			},
		}

		body := []byte(`{"model":"gpt-4o","service_tier":"flex"}`)
		request := &llm.Request{
			Model:      "gpt-4o",
			RawRequest: &httpclient.Request{Body: body},
		}
		result := runRequestRewrite(t, rules, request)
		assert.JSONEq(t, `{"model":"gpt-4o","service_tier":"default"}`, string(result.RawRequest.Body))
	})

	t.Run("unmatched value passes through", func(t *testing.T) {
		rules := []*ent.RequestRewriteRule{
			{
				ID:           1,
				ModelPattern: "*",
				FieldMaps: []objects.RequestRewriteFieldMap{
					{Path: "reasoning_effort", Values: []objects.RequestRewriteValueMap{{From: "high", To: "xhigh"}}},
				},
			},
		}

		request := &llm.Request{Model: "gpt-4o", ReasoningEffort: "low"}
		result := runRequestRewrite(t, rules, request)
		assert.Equal(t, "low", result.ReasoningEffort)
	})
}

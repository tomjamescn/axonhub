package orchestrator

import (
	"context"
	"strconv"

	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"

	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/log"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/server/biz"
	"github.com/looplj/axonhub/llm"
	"github.com/looplj/axonhub/llm/pipeline"
)

// requestRewriteRulesProvider returns the currently enabled rewrite rules.
type requestRewriteRulesProvider func(ctx context.Context) []*ent.RequestRewriteRule

// requestRewriteProviderFromService adapts the biz service to the provider
// function, returning nil for a nil concrete service.
func requestRewriteProviderFromService(svc *biz.RequestRewriteRuleService) requestRewriteRulesProvider {
	if svc == nil {
		return nil
	}

	return svc.EnabledRules
}

// applyRequestRewrite applies request rewrite rules to the unified request.
// It runs before applyModelMapping so rules match the model name exactly as
// the client sent it, decoupled from AxonHub's internal model mapping. If a
// rule rewrites the model field, the downstream flows (model access check,
// model mapping, channel selection) operate on the rewritten value.
func applyRequestRewrite(provider requestRewriteRulesProvider) pipeline.Middleware {
	if provider == nil {
		return requestRewriteNoopMiddleware()
	}

	return pipeline.OnLlmRequest("apply-request-rewrite", func(ctx context.Context, llmRequest *llm.Request) (*llm.Request, error) {
		if llmRequest == nil || llmRequest.Model == "" {
			return llmRequest, nil
		}

		rules := provider(ctx)
		if len(rules) == 0 {
			return llmRequest, nil
		}

		fieldMaps := biz.MatchedFields(rules, llmRequest.Model)
		if len(fieldMaps) == 0 {
			return llmRequest, nil
		}

		changed := rewriteRequestFields(ctx, llmRequest, fieldMaps)
		if len(changed) > 0 {
			log.Debug(ctx, "applied request rewrite rules",
				log.String("model", llmRequest.Model),
				log.Any("changed_fields", changed),
			)
		}

		return llmRequest, nil
	})
}

func requestRewriteNoopMiddleware() pipeline.Middleware {
	return pipeline.OnLlmRequest("apply-request-rewrite", func(ctx context.Context, llmRequest *llm.Request) (*llm.Request, error) {
		return llmRequest, nil
	})
}

// rewriteRequestFields applies the merged field maps to the request. It
// returns the list of changed field paths.
func rewriteRequestFields(ctx context.Context, llmRequest *llm.Request, fieldMaps []objects.RequestRewriteFieldMap) []string {
	changed := make([]string, 0, len(fieldMaps))
	for _, field := range fieldMaps {
		if field.Path == "" || len(field.Values) == 0 {
			continue
		}

		original, exists := readRequestField(llmRequest, field.Path)
		if !exists {
			continue
		}

		replacement, ok := objects.ApplyFieldMapsTo(fieldMaps, field.Path, original)
		if !ok || replacement == original {
			continue
		}

		if err := setRequestField(llmRequest, field.Path, replacement); err != nil {
			log.Warn(ctx, "failed to apply request rewrite",
				log.String("field", field.Path),
				log.String("from", original),
				log.String("to", replacement),
				log.Cause(err),
			)

			continue
		}

		changed = append(changed, field.Path)
	}

	return changed
}

// readRequestField returns the current string form of a request field. The
// boolean result is false when the field is not present in the request.
func readRequestField(llmRequest *llm.Request, path string) (string, bool) {
	switch path {
	case "model":
		return llmRequest.Model, llmRequest.Model != ""
	case "reasoning_effort":
		return llmRequest.ReasoningEffort, llmRequest.ReasoningEffort != ""
	case "reasoning_budget":
		if llmRequest.ReasoningBudget != nil {
			return strconv.FormatInt(*llmRequest.ReasoningBudget, 10), true
		}
	case "temperature":
		if llmRequest.Temperature != nil {
			return strconv.FormatFloat(*llmRequest.Temperature, 'f', -1, 64), true
		}
	case "top_p":
		if llmRequest.TopP != nil {
			return strconv.FormatFloat(*llmRequest.TopP, 'f', -1, 64), true
		}
	case "max_tokens":
		if llmRequest.MaxTokens != nil {
			return strconv.FormatInt(*llmRequest.MaxTokens, 10), true
		}
	case "max_completion_tokens":
		if llmRequest.MaxCompletionTokens != nil {
			return strconv.FormatInt(*llmRequest.MaxCompletionTokens, 10), true
		}
	case "stream":
		if llmRequest.Stream != nil {
			return strconv.FormatBool(*llmRequest.Stream), true
		}
	case "user":
		if llmRequest.User != nil {
			return *llmRequest.User, true
		}
	case "seed":
		if llmRequest.Seed != nil {
			return strconv.FormatInt(*llmRequest.Seed, 10), true
		}
	}

	// Fall back to the raw client body for fields not represented in the
	// unified request struct.
	if llmRequest.RawRequest != nil && len(llmRequest.RawRequest.Body) > 0 {
		if value := gjson.GetBytes(llmRequest.RawRequest.Body, path); value.Exists() {
			return value.String(), true
		}
	}

	return "", false
}

// setRequestField writes the replacement value back to the request. An empty
// replacement clears the field. Fields represented in the unified struct are
// updated there (so every outbound transformer sees the change); the raw
// client body is kept in sync so body pass-through does not replay the old
// value.
func setRequestField(llmRequest *llm.Request, path, value string) error {
	switch path {
	case "model":
		llmRequest.Model = value
	case "reasoning_effort":
		llmRequest.ReasoningEffort = value
	case "reasoning_budget":
		if value == "" {
			llmRequest.ReasoningBudget = nil

			return syncRawBodyField(llmRequest, path, value)
		}
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		llmRequest.ReasoningBudget = &n
	case "temperature":
		if value == "" {
			llmRequest.Temperature = nil

			return syncRawBodyField(llmRequest, path, value)
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		llmRequest.Temperature = &f
	case "top_p":
		if value == "" {
			llmRequest.TopP = nil

			return syncRawBodyField(llmRequest, path, value)
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return err
		}
		llmRequest.TopP = &f
	case "max_tokens":
		if value == "" {
			llmRequest.MaxTokens = nil

			return syncRawBodyField(llmRequest, path, value)
		}
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		llmRequest.MaxTokens = &n
	case "max_completion_tokens":
		if value == "" {
			llmRequest.MaxCompletionTokens = nil

			return syncRawBodyField(llmRequest, path, value)
		}
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		llmRequest.MaxCompletionTokens = &n
	case "stream":
		if value == "" {
			llmRequest.Stream = nil

			return syncRawBodyField(llmRequest, path, value)
		}
		b, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		llmRequest.Stream = &b
	case "user":
		if value == "" {
			llmRequest.User = nil

			return syncRawBodyField(llmRequest, path, value)
		}
		llmRequest.User = &value
	case "seed":
		if value == "" {
			llmRequest.Seed = nil

			return syncRawBodyField(llmRequest, path, value)
		}
		n, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return err
		}
		llmRequest.Seed = &n
	default:
		return syncRawBodyField(llmRequest, path, value)
	}

	return syncRawBodyField(llmRequest, path, value)
}

// syncRawBodyField patches the raw client body so body pass-through and
// persistence observe the rewritten value.
func syncRawBodyField(llmRequest *llm.Request, path, value string) error {
	if llmRequest.RawRequest == nil || len(llmRequest.RawRequest.Body) == 0 {
		return nil
	}

	if value == "" {
		body, err := sjson.DeleteBytes(llmRequest.RawRequest.Body, path)
		if err != nil {
			return err
		}
		llmRequest.RawRequest.Body = body

		return nil
	}

	body, err := sjson.SetBytes(llmRequest.RawRequest.Body, path, value)
	if err != nil {
		return err
	}
	llmRequest.RawRequest.Body = body

	return nil
}

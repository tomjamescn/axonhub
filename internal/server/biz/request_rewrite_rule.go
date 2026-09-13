package biz

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/authz"
	"github.com/looplj/axonhub/internal/ent"
	"github.com/looplj/axonhub/internal/ent/requestrewriterule"
	"github.com/looplj/axonhub/internal/log"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/pkg/watcher"
	"github.com/looplj/axonhub/internal/pkg/xcache"
	"github.com/looplj/axonhub/internal/pkg/xcache/live"
	"github.com/looplj/axonhub/internal/pkg/xerrors"
	"github.com/looplj/axonhub/internal/pkg/xregexp"
)

type RequestRewriteRuleServiceParams struct {
	fx.In

	CacheConfig xcache.Config
	Ent         *ent.Client
}

// RequestRewriteRuleService manages request rewrite rules and keeps an
// in-memory index of all enabled rules for fast per-request matching.
type RequestRewriteRuleService struct {
	*AbstractService

	enabledRulesCache *live.Cache[[]*ent.RequestRewriteRule]
	ruleNotifier      watcher.Notifier[live.CacheEvent[struct{}]]
}

func NewRequestRewriteRuleService(params RequestRewriteRuleServiceParams) *RequestRewriteRuleService {
	svc := &RequestRewriteRuleService{
		AbstractService: &AbstractService{
			db: params.Ent,
		},
	}

	cacheMode := params.CacheConfig.Mode
	if cacheMode == "" {
		cacheMode = xcache.ModeMemory
	}

	watcherMode := cacheMode
	if watcherMode == xcache.ModeTwoLevel {
		watcherMode = watcher.ModeRedis
	}

	modelNotifier, err := watcher.NewWatcherFromConfig[live.CacheEvent[struct{}]](watcher.Config{
		Mode:  watcherMode,
		Redis: params.CacheConfig.Redis,
	}, watcher.WatcherFromConfigOptions{
		RedisChannel: "axonhub:cache:request_rewrite_rules",
		Buffer:       32,
	})
	if err != nil {
		panic(fmt.Errorf("request rewrite rule watcher init failed: %w", err))
	}

	svc.ruleNotifier = modelNotifier
	svc.enabledRulesCache = live.NewCache(live.Options[[]*ent.RequestRewriteRule]{
		Name:            "axonhub:enabled_request_rewrite_rules",
		InitialValue:    []*ent.RequestRewriteRule{},
		RefreshInterval: 30 * time.Second,
		DebounceDelay:   500 * time.Millisecond,
		RefreshFunc:     svc.onEnabledRulesRefreshed,
		Watcher:         modelNotifier,
	})

	if err := svc.enabledRulesCache.Load(context.Background(), true); err != nil {
		panic(fmt.Errorf("request rewrite rule cache initial load failed: %w", err))
	}

	return svc
}

func (svc *RequestRewriteRuleService) Stop() {
	if svc.enabledRulesCache != nil {
		svc.enabledRulesCache.Stop()
	}
}

func (svc *RequestRewriteRuleService) onEnabledRulesRefreshed(ctx context.Context, _ []*ent.RequestRewriteRule, lastUpdate time.Time) ([]*ent.RequestRewriteRule, time.Time, bool, error) {
	ctx = authz.WithSystemBypass(ctx, "request-rewrite-rule-cache")
	client := svc.entFromContext(ctx)

	rules, err := client.RequestRewriteRule.Query().
		Where(
			requestrewriterule.StatusEQ(requestrewriterule.StatusEnabled),
		).
		Order(ent.Asc(requestrewriterule.FieldID)).
		All(ctx)
	if err != nil {
		return nil, lastUpdate, false, err
	}

	return rules, lastUpdate, true, nil
}

func (svc *RequestRewriteRuleService) asyncReloadRules() {
	if svc.ruleNotifier == nil {
		return
	}

	if err := svc.ruleNotifier.Notify(context.Background(), live.NewForceRefreshEvent[struct{}]()); err != nil {
		log.Warn(context.Background(), "request rewrite rule cache watcher notify failed", log.Cause(err))
	}
}

// EnabledRules returns the cached enabled rules.
func (svc *RequestRewriteRuleService) EnabledRules(ctx context.Context) []*ent.RequestRewriteRule {
	if svc.enabledRulesCache != nil {
		return svc.enabledRulesCache.GetData()
	}

	rules, err := svc.entFromContext(ctx).RequestRewriteRule.Query().
		Where(requestrewriterule.StatusEQ(requestrewriterule.StatusEnabled)).
		Order(ent.Asc(requestrewriterule.FieldID)).
		All(ctx)
	if err != nil {
		log.Warn(ctx, "failed to query enabled request rewrite rules", log.Cause(err))

		return nil
	}

	return rules
}

// MatchedFields collects the field maps of all rules whose model pattern
// matches the given (client original) model name. When multiple rules match
// the same field and source value, the first (lowest ID) mapping wins.
// Returns nil when no rule matches.
func MatchedFields(rules []*ent.RequestRewriteRule, model string) []objects.RequestRewriteFieldMap {
	merged := make([]objects.RequestRewriteFieldMap, 0, len(rules))
	for _, rule := range rules {
		if xregexp.MatchString(rule.ModelPattern, model) {
			merged = append(merged, rule.FieldMaps...)
		}
	}

	return merged
}

func (svc *RequestRewriteRuleService) ValidateRuleInput(modelPattern string, fieldMaps []objects.RequestRewriteFieldMap) error {
	if modelPattern == "" {
		return objects.ErrEmptyRequestRewriteModel
	}

	if err := xregexp.ValidateRegex(modelPattern); err != nil {
		return err
	}

	return objects.ValidateFieldMaps(fieldMaps)
}

func (svc *RequestRewriteRuleService) CreateRule(ctx context.Context, input ent.CreateRequestRewriteRuleInput) (*ent.RequestRewriteRule, error) {
	if err := svc.ValidateRuleInput(input.ModelPattern, input.FieldMaps); err != nil {
		return nil, err
	}

	existing, err := svc.entFromContext(ctx).RequestRewriteRule.Query().
		Where(requestrewriterule.Name(input.Name)).
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check existing request rewrite rule: %w", err)
	}

	if existing != nil {
		return nil, xerrors.DuplicateNameError("request rewrite rule", input.Name)
	}

	rule, err := svc.entFromContext(ctx).RequestRewriteRule.Create().
		SetInput(input).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create request rewrite rule: %w", err)
	}

	svc.asyncReloadRules()

	return rule, nil
}

func (svc *RequestRewriteRuleService) UpdateRule(ctx context.Context, id int, input *ent.UpdateRequestRewriteRuleInput) (*ent.RequestRewriteRule, error) {
	current, err := svc.entFromContext(ctx).RequestRewriteRule.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to query request rewrite rule: %w", err)
	}

	modelPattern := current.ModelPattern
	if input.ModelPattern != nil {
		modelPattern = *input.ModelPattern
	}

	fieldMaps := current.FieldMaps
	if input.FieldMaps != nil {
		fieldMaps = input.FieldMaps
	}

	if err := svc.ValidateRuleInput(modelPattern, fieldMaps); err != nil {
		return nil, err
	}

	rule, err := svc.entFromContext(ctx).RequestRewriteRule.UpdateOneID(id).
		SetInput(*input).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update request rewrite rule: %w", err)
	}

	svc.asyncReloadRules()

	return rule, nil
}

func (svc *RequestRewriteRuleService) DeleteRule(ctx context.Context, id int) error {
	if err := svc.entFromContext(ctx).RequestRewriteRule.DeleteOneID(id).Exec(ctx); err != nil {
		return fmt.Errorf("failed to delete request rewrite rule: %w", err)
	}

	svc.asyncReloadRules()

	return nil
}

func (svc *RequestRewriteRuleService) UpdateRuleStatus(ctx context.Context, id int, status requestrewriterule.Status) (*ent.RequestRewriteRule, error) {
	rule, err := svc.entFromContext(ctx).RequestRewriteRule.UpdateOneID(id).
		SetStatus(status).
		Save(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to update request rewrite rule status: %w", err)
	}

	svc.asyncReloadRules()

	return rule, nil
}

func (svc *RequestRewriteRuleService) BulkDeleteRules(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	if _, err := svc.entFromContext(ctx).RequestRewriteRule.Delete().
		Where(requestrewriterule.IDIn(ids...)).
		Exec(ctx); err != nil {
		return fmt.Errorf("failed to bulk delete request rewrite rules: %w", err)
	}

	svc.asyncReloadRules()

	return nil
}

func (svc *RequestRewriteRuleService) BulkDisableRules(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	if _, err := svc.entFromContext(ctx).RequestRewriteRule.Update().
		Where(requestrewriterule.IDIn(ids...)).
		SetStatus(requestrewriterule.StatusDisabled).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to bulk disable request rewrite rules: %w", err)
	}

	svc.asyncReloadRules()

	return nil
}

func (svc *RequestRewriteRuleService) BulkEnableRules(ctx context.Context, ids []int) error {
	if len(ids) == 0 {
		return nil
	}

	if _, err := svc.entFromContext(ctx).RequestRewriteRule.Update().
		Where(requestrewriterule.IDIn(ids...)).
		SetStatus(requestrewriterule.StatusEnabled).
		Save(ctx); err != nil {
		return fmt.Errorf("failed to bulk enable request rewrite rules: %w", err)
	}

	svc.asyncReloadRules()

	return nil
}

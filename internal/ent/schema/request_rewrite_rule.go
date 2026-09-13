package schema

import (
	"entgo.io/contrib/entgql"
	"entgo.io/ent"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"

	"github.com/looplj/axonhub/internal/ent/schema/schematype"
	"github.com/looplj/axonhub/internal/objects"
	"github.com/looplj/axonhub/internal/scopes"
)

// RequestRewriteRule holds field-value rewrite rules applied to client
// requests before model routing. A rule matches requests by model pattern
// (exact or regex, applied to the model name exactly as the client sends it).
type RequestRewriteRule struct {
	ent.Schema
}

func (RequestRewriteRule) Mixin() []ent.Mixin {
	return []ent.Mixin{
		TimeMixin{},
		schematype.SoftDeleteMixin{},
	}
}

func (RequestRewriteRule) Indexes() []ent.Index {
	return []ent.Index{
		// Unique rule name per user
		index.Fields("user_id", "name", "deleted_at").
			StorageKey("request_rewrite_rules_by_user_name").
			Unique(),
		// Accelerates loading the enabled rules
		index.Fields("status"),
	}
}

func (RequestRewriteRule) Fields() []ent.Field {
	return []ent.Field{
		field.Int("user_id").
			Optional().
			Immutable().
			Comment("Owner of this rule").
			Annotations(
				entgql.Skip(entgql.SkipMutationUpdateInput),
			),
		field.String("name").
			NotEmpty().
			Comment("Rule name, unique per user"),
		field.String("description").
			Default("").
			Comment("Rule description"),
		field.String("model_pattern").
			NotEmpty().
			Comment("Exact model name or regex pattern (applied to the client's original model name)"),
		field.Enum("status").
			Values("enabled", "disabled", "archived").
			Default("disabled").
			Annotations(entgql.Skip(entgql.SkipMutationCreateInput)),
		field.JSON("field_maps", []objects.RequestRewriteFieldMap{}).
			Default([]objects.RequestRewriteFieldMap{}).
			Comment("Field value maps: first matching source value wins; empty replacement clears the field"),
	}
}

func (RequestRewriteRule) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entgql.QueryField(),
		entgql.RelayConnection(),
		entgql.Mutations(entgql.MutationCreate(), entgql.MutationUpdate()),
	}
}

func (RequestRewriteRule) Policy() ent.Policy {
	return scopes.Policy{
		Query: scopes.QueryPolicy{
			scopes.APIKeyScopeQueryRule(scopes.ScopeReadChannels),
			scopes.OwnerRule(),
			scopes.UserReadScopeRule(scopes.ScopeReadChannels),
		},
		Mutation: scopes.MutationPolicy{
			scopes.OwnerRule(),
			scopes.UserWriteScopeRule(scopes.ScopeWriteChannels),
		},
	}
}

# Request Rewrite Rules

AxonHub provides request rewrite rules that let you change the value of request body fields per model, before a request is routed to a channel. This is useful when a client sends non-standard field values that a specific upstream model or channel does not accept — for example, a client sending `"reasoning_effort": "xhigh"` to a channel whose model only supports `high`, `medium`, or `low`.

## Overview

A Request Rewrite Rule defines:

1. **Which model** it applies to — an exact model name or a regex pattern, matched against the model name exactly as the client sends it.
2. **Which field values to rewrite** — one or more request body fields, each with a list of `current value → new value` mappings.

When a request arrives, enabled rules are looked up by model, and each mapped field value in the request is replaced.

## How It Works

Rewrite rules are the **first gate** a client request passes through. They act as a decoupled outer filter on the raw client request:

- They run as soon as the request is parsed, **before** model mapping, model access checks, and channel selection.
- Model patterns match the model name **exactly as the client sent it**.
- They are decoupled from all of AxonHub's downstream routing logic — unless a rule rewrites the `model` field itself.

```
┌──────────────────┐
│   Client Request │  e.g. {"model":"gpt-4o","reasoning_effort":"xhigh"}
└────────┬─────────┘
         │
         ▼
┌──────────────────────────────────────────────┐
│           Request Rewrite Rules              │
│  1. Load enabled rules (cached)              │
│  2. Match rules by the client's original     │
│     model name                               │
│  3. Replace mapped field values in the       │
│     request body                             │
└────────┬─────────────────────────────────────┘
         │
         ▼
┌────────────────────────┐
│ API key model mapping  │  (existing feature)
└────────┬───────────────┘
         │
         ▼
┌─────────────────┐
│  Routed request │  e.g. {"model":"gpt-4o","reasoning_effort":"high"}
└─────────────────┘
```

## Core Concepts

### Rule Structure

| Field | Type | Description |
|-------|------|-------------|
| **Name** | String | Unique identifier for the rule (per user) |
| **Description** | String | Optional description |
| **Model Pattern** | String | Exact model name or regex, matched against the client's original model name |
| **Status** | Enum | `enabled`, `disabled`, or `archived` |
| **Field Maps** | List | One entry per rewritten field |

### Field Map

Each field map has a `path` (the request body field name) and a list of value mappings:

| Field | Description |
|-------|-------------|
| **path** | Request body field, e.g. `reasoning_effort`, `temperature`, `service_tier` |
| **from** | The current value to match. An empty value matches an absent/empty field. |
| **to** | The replacement value. An empty value removes the field from the request. |

For a given field, the **first matching `from` value wins**. A current value with no mapping passes through unchanged. When multiple rules match the same field and value, the first matching rule (lowest ID) wins.

### Model Pattern Matching

The pattern is matched against the model name exactly as the client sent it — before API key profile model mapping, model access checks, and channel selection.

- Exact names match directly: `gpt-4o`
- Patterns containing regex characters are compiled and anchored as full regexes: `gpt-.*`, `claude-.*-haiku.*`
- `*` alone matches every model

If a rule rewrites the `model` field itself, downstream flows (model access check, model mapping, channel selection) operate on the rewritten value.

## Creating Rules

### Via Admin UI

1. Navigate to **Request Rewrite Rules** (改写规则) in the sidebar
2. Click **Create Rule**
3. Configure the rule:
   - Enter a unique **Name**
   - Add an optional **Description**
   - Set the **Model Pattern** (exact name or regex)
   - Add one or more **Field Mappings**: field path plus `current value → new value` pairs
4. Click **Create**
5. Enable the rule with the status toggle

## Usage Examples

### Example 1: Normalize non-standard reasoning effort

A client sends `"reasoning_effort": "xhigh"`, but the channel's model only accepts `high`.

**Rule:**
- Model Pattern: `gpt-4o`
- Field `reasoning_effort`: `xhigh → high`

**Request before:**
```json
{ "model": "gpt-4o", "reasoning_effort": "xhigh" }
```

**Request sent upstream:**
```json
{ "model": "gpt-4o", "reasoning_effort": "high" }
```

### Example 2: Remove a field for a model family

Some channels reject an unknown field. Remove it for all matching models:

- Model Pattern: `deepseek-.*`
- Field `service_tier`: `flex → *(empty)*`

The `service_tier` field is removed from the outgoing request.

### Example 3: Pin temperature for a model

- Model Pattern: `claude-.*`
- Field `temperature`: `0.9 → 0.5`

All requests to Claude models get `temperature` pinned to `0.5`.

## Rule Management

### Status Management

| Status | Description |
|--------|-------------|
| **enabled** | Rule is active and applied to requests |
| **disabled** | Rule is inactive but can be re-enabled |
| **archived** | Rule is soft-deleted and hidden from lists |

### Bulk Operations

- **Bulk Enable** / **Bulk Disable** / **Bulk Delete** multiple rules from the table selection.

### Cache Behavior

Enabled rules are cached in memory for performance:

- The enabled-rules index refreshes every 30 seconds
- Creating, updating, or deleting a rule triggers an async cache reload
- Model pattern compilation is cached separately

## Best Practices

1. **Use exact model names** when possible — they are cheapest and unambiguous.
2. **Match the client's original model name** — rules run before API key model mapping, so use the name clients actually send.
3. **Document the `from` values** you expect — unmatched values pass through unchanged, so typos in `from` silently do nothing.
4. **Keep mappings minimal** — each rule adds a field scan per request.

## Common Issues

### Q: Why isn't my rule applying?

1. Is the rule **enabled**?
2. Does the **model pattern** match the model name the client sends? (The raw client model, before any API key profile mapping.)
3. Does the request actually contain the field with the exact `from` value? Values are matched exactly (case-sensitive).

### Q: What if multiple rules match the same field and value?

The first matching rule (lowest ID) wins for that value.

### Q: Can I rewrite nested fields?

Field paths use the top-level request body field name (e.g. `reasoning_effort`). Supported fields are applied to the unified request model so all outbound API formats see the change; other top-level fields are patched on the raw client body.

### Q: Does rewriting affect what the client sees or what is logged?

Rewrites happen before routing and logging. Request logs and traces record the rewritten values, and body pass-through channels send the rewritten body.

## Related Documentation

- [Request Override](request-override.md) — channel-level outbound request/header override
- [API Key Profiles](api-key-profiles.md) — model mapping and per-token routing
- [Model Management](model-management.md) — models and associations
- [Channel Management](channel-management.md)

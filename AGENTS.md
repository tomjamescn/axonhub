# AGENTS.md

This file provides guidance to AI coding assistants when working with code in this repository.

> **Detailed rules are split into focused files under `.agent/rules/`**. See [Rules Index](#rules-index) below.

## Global Rules

1. Do NOT run lint or build commands unless explicitly requested by the user.
2. Do NOT restart the development server — it's already started and managed.
3. All summary files should be stored in `.agent/summary` directory if available.

## Configuration

- Backend API: port 8090, Frontend dev server: port 5173 (proxies to backend).
- Configuration: `conf/conf.go` (YAML + env var), SQLite by default.

## Project Overview

AxonHub is an all-in-one AI development platform that serves as a unified API gateway for multiple AI providers. It provides OpenAI and Anthropic-compatible API interfaces with automatic request transformation, enabling seamless communication between clients and various AI providers through a sophisticated bidirectional data transformation pipeline.

## Technology Stack

- **Backend**: Go 1.26.0+ with Gin, Ent ORM, gqlgen, FX
- **Frontend**: React 19 + TypeScript, TanStack Router/Query, Zustand, Tailwind CSS

## Backend Structure

- `cmd/axonhub/main.go` — Application entry point
- `internal/server/` — HTTP server and route handling with Gin
- `internal/server/biz/` — Core business logic and services
- `internal/server/api/` — REST and GraphQL API handlers
- `internal/server/gql/` — GraphQL schema and resolvers
- `internal/ent/` — Ent ORM for database operations
- `internal/ent/schema/` — Database schema definitions
- `internal/contexts/` — Context handling utilities
- `internal/pkg/` — Shared utilities (xerrors, xjson, xcache, xfile, xcontext, etc.)
- `internal/scopes/` — Permission system with role-based access control
- `llm/` — LLM utilities, transformers, and pipeline processing (separate Go module)
- `llm/pipeline/` — Pipeline processing architecture
- `conf/conf.go` — Configuration loading and validation

## Go Modules

- The repository root (`/`) is the main Go module: `github.com/looplj/axonhub`.
- `llm/` is a separate Go module: `github.com/looplj/axonhub/llm`.

### `llm/` Module Notes

- `llm/` is an independent module. Always run Go commands from the `llm/` directory (e.g., `cd llm && go test ./...`).
- Running `go test ./llm/...` from repo root will fail with module boundary errors.

## Frontend Structure

- `frontend/src/routes/` — TanStack Router file-based routing
- `frontend/src/gql/` — GraphQL API communication
- `frontend/src/features/` — Feature-based component organization
- `frontend/src/components/` — Reusable shared components
- `frontend/src/hooks/` — Custom shared hooks
- `frontend/src/stores/` — Zustand state management
- `frontend/src/locales/` — i18n support (en.json, zh.json)
- `frontend/src/lib/` — Core utilities (API client, i18n, permissions, utils)
- `frontend/src/utils/` — Domain-specific utilities (date, format, error handling)
- `frontend/src/config/` — App configuration
- `frontend/src/context/` — React context providers

## Rules Index

All detailed rules are in `.agent/rules/`:

| File | Scope | Description |
|------|-------|-------------|
| [go-general.md](.agent/rules/go-general.md) | `**/*.go` | Go 通用约定、错误处理、依赖注入、开发命令约束 |
| [ent-graphql.md](.agent/rules/ent-graphql.md) | `internal/ent/schema/**/*.go`, `internal/server/gql/**/*.go`, `internal/server/gql/**/*.graphql`, `gqlgen.yml` | Ent、GraphQL、代码生成、schema 变更规则 |
| [database-indexes.md](.agent/rules/database-indexes.md) | `internal/ent/schema/**/*.go` | 数据库索引设计、命名、跨方言兼容与迁移验证规则 |
| [biz-services.md](.agent/rules/biz-services.md) | `internal/server/biz/**/*.go` | Biz service、上下文取值、事务与级联删除规则 |
| [cache-compat.md](.agent/rules/cache-compat.md) | `**/*.go` | 缓存结构兼容性与升级安全规则 |
| [frontend-general.md](.agent/rules/frontend-general.md) | `frontend/**/*.ts`, `frontend/**/*.tsx` | 前端通用开发约定、GraphQL 数据约束、页面作用域 |
| [frontend-i18n.md](.agent/rules/frontend-i18n.md) | `frontend/src/**/*.ts`, `frontend/src/**/*.tsx`, `frontend/src/locales/*.json` | i18n 与货币格式规则 |
| [frontend-ui.md](.agent/rules/frontend-ui.md) | `frontend/**/*.tsx` | 前端 UI 组件使用规则 |
| [e2e.md](.agent/rules/e2e.md) | `frontend/tests/**/*.ts` | E2E testing rules |
| [docs.md](.agent/rules/docs.md) | `docs/**/*.md` | Documentation rules |
| [workflows/add-channel.md](.agent/rules/workflows/add-channel.md) | Manual | Workflow for adding a new channel |

## Go Modules (complete list)

Besides the two listed above, the repo contains four more independent Go modules — always run `go` commands from the owning module directory:

- `integration_test/openai/`, `integration_test/anthropic/`, `integration_test/gemini/` — provider integration test modules; need a running AxonHub plus a real provider API key. Not covered by `make test-backend-all`.
- `cmd/schema/` — config JSON-schema generator (`make generate-schema`).
- `examples/openapi/` — example clients.

## Codegen

- After any Ent schema (`internal/ent/schema/`) or GraphQL schema (`internal/server/gql/**/*.graphql`) change, run `make generate`.
- `internal/ent/migrate/schema.go` is generated output — edit the owning schema's `Indexes()`/fields, never the generated file.
- GraphQL: edit `*.graphql` first; `ent.graphql` is generated. New GraphQL structs need a mapping in `gqlgen.yml`.

## Sub-Path / URL Prefix Deployment

AxonHub can be served under a custom URL prefix (e.g. `/llmproxy`) for reverse-proxy deployments. This is a **frontend build-time** concern only — the backend is unchanged; the reverse proxy strips the prefix before forwarding to the backend root.

- `Dockerfile` exposes `ARG BASE_PATH` (default `/`), which sets `VITE_BASE_PATH` and `VITE_API_BASE_PATH` for the frontend build stage.
- `frontend/vite.config.ts` reads those env vars: sets Vite `base` and injects the build-time global `__API_BASE_PATH__` (declared in `frontend/src/vite-env.d.ts`).
- Every API/asset path derives the prefix from `__API_BASE_PATH__`: `apiRequest` (via `API_BASE_URL`), `GRAPHQL_ENDPOINT`, TanStack Router `basepath`, the playground chat endpoint, request-detail fetches, and the sign-in redirects. With the default `/` the prefix is empty and behavior is unchanged.
- If you add a new raw `fetch`/`WebSocket`/`EventSource` call (one that does not go through `apiRequest` or `GRAPHQL_ENDPOINT`), prefix it with `${__API_BASE_PATH__ || ''}` or sub-path deploys will break.
- Build with a prefix: `docker build --build-arg BASE_PATH=/llmproxy -t axonhub:llmproxy .`
- The reverse proxy must strip the prefix, e.g. nginx: `location /llmproxy/ { proxy_pass http://axonhub:8090/; }` (the trailing slash on `proxy_pass` drops the prefix). Keep `Upgrade`/`Connection` headers for SSE/WebSocket.

## Developer Commands

- Backend tests (root + `llm` modules): `make test-backend-all`.
- Lint: `make lint` (golangci-lint over all Go modules).
- Frontend unit tests: `cd frontend && pnpm test:unit` (plain `node --test`, no framework).
- E2E: `make e2e-test` (Playwright) — starts a throwaway backend on port **8099** with a fresh DB; env `AXONHUB_E2E_DB_TYPE` for MySQL/Postgres. E2E login: `my@example.com` / `pwd123456`.
- Migration tests: `make migration-test TAG=vX.Y.Z`, `make migration-test-all`.
- Docker build: `docker build --build-arg GOPROXY=https://goproxy.cn -t <tag> .` when the build container cannot reach `proxy.golang.org`.
- Docker build with URL prefix: `docker build --build-arg BASE_PATH=/llmproxy -t <tag> .` (see [Sub-Path / URL Prefix Deployment](#sub-path--url-prefix-deployment)).

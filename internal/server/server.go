package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/fx"

	"github.com/looplj/axonhub/internal/log"
	"github.com/looplj/axonhub/internal/server/api"
	"github.com/looplj/axonhub/internal/server/backup"
	"github.com/looplj/axonhub/internal/server/biz"
	"github.com/looplj/axonhub/internal/server/dependencies"
	"github.com/looplj/axonhub/internal/server/gc"
	"github.com/looplj/axonhub/internal/server/gql"
	"github.com/looplj/axonhub/internal/server/gql/openapi"
	"github.com/looplj/axonhub/internal/server/middleware"
	"github.com/looplj/axonhub/internal/server/orchestrator"
	"github.com/looplj/axonhub/internal/server/scheduler"
	"github.com/looplj/axonhub/internal/server/video_storage"
	"github.com/looplj/axonhub/internal/tracing"
)

func New(config Config) *Server {
	if !config.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	// Gin trusts all proxies by default. Never inherit that unsafe default:
	// forwarded client IP headers must only be honored for explicitly configured
	// proxy networks, otherwise IP access controls and API-key IP restrictions
	// can be bypassed with a forged request header.
	if err := engine.SetTrustedProxies(config.TrustedProxies); err != nil {
		panic(fmt.Errorf("invalid server.trusted_proxies: %w", err))
	}

	// Set max multipart memory for file uploads (e.g., backup restore).
	// Default 32 MB may be insufficient for large backup files.
	if config.MaxMultipartMemory > 0 {
		engine.MaxMultipartMemory = int64(config.MaxMultipartMemory)
	}

	engine.Use(middleware.Recovery())

	return &Server{
		Config: config,
		Engine: engine,
	}
}

type Server struct {
	*gin.Engine

	Config Config
	server *http.Server
	addr   string
}

func (srv *Server) Run() error {
	log.Info(context.Background(), "run server",
		log.String("name", srv.Config.Name),
		log.String("host", srv.Config.Host),
		log.Int("port", srv.Config.Port),
	)
	addr := fmt.Sprintf("%s:%d", srv.Config.Host, srv.Config.Port)
	srv.server = &http.Server{
		Addr:         addr,
		Handler:      srv.handler(),
		ReadTimeout:  srv.Config.ReadTimeout,
		WriteTimeout: max(srv.Config.RequestTimeout, srv.Config.LLMRequestTimeout),
	}
	srv.addr = addr

	err := srv.server.ListenAndServe()
	if err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return err
	}

	return nil
}

// handler returns the root http.Handler. When server.base_path is configured
// (e.g. /llmproxy), every incoming request whose path is under that prefix
// gets the prefix stripped before reaching the Gin engine, so the whole
// application (API, GraphQL, static frontend) also works behind reverse
// proxies that forward the URL without rewriting it. Requests without the
// prefix are served unchanged, so direct access to the port keeps working.
func (srv *Server) handler() http.Handler {
	base := normalizeBasePath(srv.Config.BasePath)
	if base == "" {
		return srv.Engine
	}

	log.Info(context.Background(), "serving under base path", log.String("base_path", base))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == base || strings.HasPrefix(path, base+"/") {
			trimmed := strings.TrimPrefix(path, base)
			if trimmed == "" {
				trimmed = "/"
			}
			r = r.Clone(r.Context())
			r.URL.Path = trimmed
			r.URL.RawPath = ""
			r.RequestURI = r.URL.RequestURI()
		}

		srv.Engine.ServeHTTP(w, r)
	})
}

// normalizeBasePath canonicalizes a configured base path: it trims whitespace
// and trailing slashes, ensures a leading slash, and returns "" for the root
// path (no prefix).
func normalizeBasePath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	p = strings.TrimRight(p, "/")
	if p == "" {
		return ""
	}
	return p
}

func (srv *Server) Shutdown(ctx context.Context) error {
	return srv.server.Shutdown(ctx)
}

func Run(opts ...fx.Option) {
	constructors := []any{
		openapi.NewGraphqlHandlers,
		gql.NewGraphqlHandlers,
		gc.NewWorker,
		New,
		NewIPAccessControlRuntime,
	}

	app := fx.New(
		append([]fx.Option{
			fx.NopLogger,
			fx.Provide(constructors...),
			dependencies.Module,
			scheduler.Module,
			biz.Module,
			orchestrator.Module,
			backup.Module,
			video_storage.Module,
			api.Module,
			fx.Provide(fx.Annotate(func(cfg Config) string { return cfg.PublicURL }, fx.ResultTags(`name:"public_url"`))),
			fx.Provide(func(cfg Config) api.SSEKeepAliveConfig {
				return api.SSEKeepAliveConfig{
					Enabled:  cfg.SSEKeepAlive.Enabled,
					Interval: cfg.SSEKeepAlive.Interval,
				}
			}),
			fx.Invoke(func(cfg log.Config) {
				log.SetGlobalConfig(cfg)
				tracing.SetupLogger(log.GetGlobalLogger())
				slog.SetDefault(log.GetGlobalLogger().AsSlog())
			}),
			fx.Invoke(func(usageLogSvc *biz.UsageLogService) {
				usageLogSvc.OnUsageLogCreated = gql.InvalidateAllTimeTokenStatsCache
			}),
			fx.Invoke(func(cfg Config) {
				if cfg.Dashboard.AllTimeTokenStatsSoftTTL > 0 && cfg.Dashboard.AllTimeTokenStatsHardTTL > 0 {
					gql.SetTokenStatsCacheTTL(cfg.Dashboard.AllTimeTokenStatsSoftTTL, cfg.Dashboard.AllTimeTokenStatsHardTTL)
				}
			}),
			fx.Invoke(func(lc fx.Lifecycle, worker *gc.Worker, s *scheduler.Scheduler) {
				lc.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						return worker.RegisterScheduledTasks(ctx, s)
					},
				})
			}),
			fx.Invoke(SetupRoutes),
		}, opts...)...,
	)
	app.Run()
}

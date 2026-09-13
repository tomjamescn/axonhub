package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func newBasePathTestServer(t *testing.T, basePath string) http.Handler {
	t.Helper()

	engine := New(Config{BasePath: basePath}).Engine
	engine.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"route": "root"})
	})
	engine.GET("/admin/system/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"route": c.Request.URL.Path})
	})

	srv := &Server{Engine: engine, Config: Config{BasePath: basePath}}
	return srv.handler()
}

func get(t *testing.T, handler http.Handler, path string) (int, string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestNormalizeBasePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "empty", in: "", want: ""},
		{name: "whitespace", in: "  ", want: ""},
		{name: "root", in: "/", want: ""},
		{name: "trailing slash", in: "/llmproxy/", want: "/llmproxy"},
		{name: "no leading slash", in: "llmproxy", want: "/llmproxy"},
		{name: "multiple trailing slashes", in: "/llmproxy///", want: "/llmproxy"},
		{name: "plain", in: "/llmproxy", want: "/llmproxy"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeBasePath(tt.in); got != tt.want {
				t.Fatalf("normalizeBasePath(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestHandlerStripsBasePath(t *testing.T) {
	t.Parallel()

	handler := newBasePathTestServer(t, "/llmproxy")

	tests := []struct {
		name    string
		path    string
		wantCode int
		wantBody string
	}{
		{name: "exact base path", path: "/llmproxy", wantCode: http.StatusOK, wantBody: `{"route":"root"}`},
		{name: "base path with slash", path: "/llmproxy/", wantCode: http.StatusOK, wantBody: `{"route":"root"}`},
		{name: "base path with sub path", path: "/llmproxy/admin/system/status", wantCode: http.StatusOK, wantBody: `{"route":"/admin/system/status"}`},
		{name: "prefix collision is not stripped", path: "/llmproxy-other/foo", wantCode: http.StatusNotFound},
		{name: "path outside base path", path: "/admin/system/status", wantCode: http.StatusOK, wantBody: `{"route":"/admin/system/status"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, body := get(t, handler, tt.path)
			if tt.wantCode != 0 && code != tt.wantCode {
				t.Fatalf("GET %s status = %d, want %d", tt.path, code, tt.wantCode)
			}
			if tt.wantBody != "" && body != tt.wantBody {
				t.Fatalf("GET %s body = %q, want %q", tt.path, body, tt.wantBody)
			}
		})
	}
}

func TestHandlerWithoutBasePath(t *testing.T) {
	t.Parallel()

	handler := newBasePathTestServer(t, "")

	code, body := get(t, handler, "/admin/system/status")
	if code != http.StatusOK {
		t.Fatalf("status = %d, want %d", code, http.StatusOK)
	}
	if body != `{"route":"/admin/system/status"}` {
		t.Fatalf("body = %q", body)
	}
}

package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func newTestRedis(t *testing.T) (*redis.Client, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	return client, mr
}

func setupRouter(middleware gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware)
	r.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return r
}

func TestNewRateLimiter_PanicOnInvalidBurst(t *testing.T) {
	client, _ := newTestRedis(t)

	tests := []struct {
		name  string
		burst int
		rate  float64
	}{
		{"zero burst", 0, 5.0},
		{"negative burst", -1, 5.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic, got none")
				}
			}()
			NewRateLimiter(client, tt.rate, tt.burst)
		})
	}
}

func TestNewRateLimiter_PanicOnInvalidRate(t *testing.T) {
	client, _ := newTestRedis(t)

	tests := []struct {
		name string
		rate float64
	}{
		{"zero rate", 0},
		{"negative rate", -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Error("expected panic, got none")
				}
			}()
			NewRateLimiter(client, tt.rate, 10)
		})
	}
}

func TestGlobalRateLimitMiddleware_AllowsWithinBurst(t *testing.T) {
	client, _ := newTestRedis(t)
	router := setupRouter(GlobalRateLimitMiddleware(client, 5, 3))

	for i := range 3 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: expected 200, got %d", i+1, w.Code)
		}
		if w.Header().Get("X-RateLimit-Limit") == "" {
			t.Errorf("request %d: missing X-RateLimit-Limit header", i+1)
		}
		if w.Header().Get("X-RateLimit-Remaining") == "" {
			t.Errorf("request %d: missing X-RateLimit-Remaining header", i+1)
		}
	}
}

func TestGlobalRateLimitMiddleware_RejectsOverBurst(t *testing.T) {
	client, _ := newTestRedis(t)
	router := setupRouter(GlobalRateLimitMiddleware(client, 1, 2))

	// Exhaust burst
	for range 2 {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.RemoteAddr = "1.2.3.4:1234"
		router.ServeHTTP(w, req)
	}

	// Next request should be rejected
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}
	if w.Header().Get("Retry-After") == "" {
		t.Error("missing Retry-After header on 429 response")
	}
}

func TestGlobalRateLimitMiddleware_DifferentIPsIndependent(t *testing.T) {
	client, _ := newTestRedis(t)
	router := setupRouter(GlobalRateLimitMiddleware(client, 60))

	// IP 1 uses its token
	w1 := httptest.NewRecorder()
	req1 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req1.RemoteAddr = "1.1.1.1:1234"
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("IP1: expected 200, got %d", w1.Code)
	}

	// IP 2 should still have its own bucket
	w2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodGet, "/test", nil)
	req2.RemoteAddr = "2.2.2.2:1234"
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Errorf("IP2: expected 200, got %d", w2.Code)
	}
}

func TestGlobalRateLimitMiddleware_RefillsOverTime(t *testing.T) {
	client, mr := newTestRedis(t)
	router := setupRouter(GlobalRateLimitMiddleware(client, 60))

	// Exhaust bucket
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)

	// Should be rejected now
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429, got %d", w.Code)
	}

	// Fast-forward 2 seconds, token should refill
	mr.FastForward(2 * time.Second)

	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 after refill, got %d", w.Code)
	}
}

func TestRateLimitMiddleware_PerRouteIsolation(t *testing.T) {
	client, _ := newTestRedis(t)
	rateLimiter := RateLimitMiddleware(client, 60)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(rateLimiter)
	r.GET("/route-a", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/route-b", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Exhaust /route-a
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/route-a", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/route-a: expected 200, got %d", w.Code)
	}

	// /route-b should still be allowed (separate bucket)
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/route-b", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("/route-b: expected 200, got %d", w.Code)
	}
}

func TestGlobalRateLimitMiddleware_FailOpenOnRedisDown(t *testing.T) {
	client, mr := newTestRedis(t)
	router := setupRouter(GlobalRateLimitMiddleware(client, 60))

	// Stop Redis
	mr.Close()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 (fail-open), got %d", w.Code)
	}
}

func TestGlobalRateLimitMiddleware_TTLExpiry(t *testing.T) {
	client, mr := newTestRedis(t)
	router := setupRouter(GlobalRateLimitMiddleware(client, 60))

	// Exhaust bucket
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)

	// Fast-forward past TTL (ceil(1/1)+1 = 2 seconds)
	mr.FastForward(3 * time.Second)

	// Key expired, should create new bucket → allowed
	w = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200 after TTL expiry, got %d", w.Code)
	}
}

func TestRateLimitMiddleware_UnknownRouteHandled(t *testing.T) {
	client, _ := newTestRedis(t)
	rateLimiter := RateLimitMiddleware(client, 60)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(rateLimiter)
	r.GET("/known", func(c *gin.Context) { c.Status(http.StatusOK) })

	// Request to unknown route — FullPath() returns ""
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/unknown-path", nil)
	req.RemoteAddr = "1.2.3.4:1234"
	r.ServeHTTP(w, req)

	// Should not panic, Gin returns 404 for unregistered routes
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown route, got %d", w.Code)
	}
}

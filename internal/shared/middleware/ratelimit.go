package middleware

import (
	"context"
	"fmt"
	"learning-go/internal/shared/common"
	"learning-go/internal/shared/logger"
	"learning-go/internal/shared/response"

	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// Lua script: atomic token bucket rate limiter.
// Uses Redis server time to avoid clock drift between app instances.
// KEYS[1] = rate limit key
// ARGV[1] = max tokens (burst)
// ARGV[2] = refill rate (tokens per second)
// Returns: {allowed, retry_after_seconds, remaining_tokens}
var tokenBucketScript = redis.NewScript(`
	local key = KEYS[1]
	local max_tokens = tonumber(ARGV[1])
	local refill_rate = tonumber(ARGV[2])

	local redis_time = redis.call("TIME")
	local now = redis_time[1] * 1000 + math.floor(redis_time[2] / 1000)

	local bucket = redis.call("HMGET", key, "tokens", "last_refill")
	local tokens = tonumber(bucket[1])
	local last_refill = tonumber(bucket[2])

	if tokens == nil then
		tokens = max_tokens
		last_refill = now
	end

	local elapsed = math.max(0, (now - last_refill) / 1000)
	tokens = math.min(max_tokens, tokens + elapsed * refill_rate)

	local ttl = math.ceil(max_tokens / refill_rate) + 1
	local allowed = 0
	local retry_after = 0

	if tokens >= 1 then
		tokens = tokens - 1
		allowed = 1
	else
		retry_after = math.ceil((1 - tokens) / refill_rate)
	end

	redis.call("HSET", key, "tokens", tokens, "last_refill", now)
	redis.call("EXPIRE", key, ttl)

	return {allowed, retry_after, math.floor(tokens)}
`)

const rateLimitKeyPrefix = "ratelimit:"

type RateLimiter struct {
	redisClient *redis.Client
	rate        float64
	burst       int
}

func NewRateLimiter(redisClient *redis.Client, requestsPerSecond float64, burst int) *RateLimiter {
	if burst <= 0 {
		panic(fmt.Sprintf("rate limiter: burst must be positive, got %d", burst))
	}
	if requestsPerSecond <= 0 {
		panic(fmt.Sprintf("rate limiter: requestsPerSecond must be positive, got %f", requestsPerSecond))
	}

	return &RateLimiter{
		redisClient: redisClient,
		rate:        requestsPerSecond,
		burst:       burst,
	}
}

type rateLimitResult struct {
	allowed    bool
	retryAfter int
	remaining  int
}

func (limiter *RateLimiter) allow(ctx context.Context, key string) rateLimitResult {
	res, err := tokenBucketScript.Run(ctx, limiter.redisClient, []string{key}, limiter.burst, limiter.rate).Int64Slice()
	if err != nil || len(res) < 3 {
		logger.Debug(ctx, "[SERVER] rate limit check failed, allowing request", zap.Error(err))
		return rateLimitResult{allowed: true, remaining: limiter.burst}
	}

	return rateLimitResult{
		allowed:    res[0] == 1,
		retryAfter: int(res[1]),
		remaining:  int(res[2]),
	}
}

// GlobalRateLimitMiddleware applies global per-IP rate limiting.
// Key: ratelimit:{ip}
func GlobalRateLimitMiddleware(redisClient *redis.Client, requestsPerSecond float64, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(redisClient, requestsPerSecond, burst)

	return func(c *gin.Context) {
		ip := common.ResolveClientIP(c.Request)
		key := rateLimitKeyPrefix + ip

		result := limiter.allow(c.Request.Context(), key)
		setRateLimitHeaders(c, limiter.burst, result)
		if !result.allowed {
			rejectRateLimit(c, ip, result.retryAfter)
			return
		}
		c.Next()
	}
}

// RateLimitMiddleware applies per-route per-IP rate limiting.
// Key: ratelimit:{route}::{ip}
// Route path is auto-detected from Gin context.
func RateLimitMiddleware(redisClient *redis.Client, requestsPerSecond float64, burst int) gin.HandlerFunc {
	limiter := NewRateLimiter(redisClient, requestsPerSecond, burst)

	return func(c *gin.Context) {
		ip := common.ResolveClientIP(c.Request)
		route := c.FullPath()
		if route == "" {
			route = "_unknown"
		}
		key := rateLimitKeyPrefix + route + "::" + ip

		result := limiter.allow(c.Request.Context(), key)
		setRateLimitHeaders(c, limiter.burst, result)
		if !result.allowed {
			rejectRateLimit(c, ip, result.retryAfter)
			return
		}
		c.Next()
	}
}

func setRateLimitHeaders(c *gin.Context, limit int, result rateLimitResult) {
	c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(result.remaining))
}

func rejectRateLimit(c *gin.Context, ip string, retryAfter int) {
	logger.Debug(c.Request.Context(), "[SERVER] rate limit exceeded", zap.String("client_ip", ip))
	if retryAfter > 0 {
		c.Header("Retry-After", strconv.Itoa(retryAfter))
	}
	response.TooManyRequests(c, "common.too_many_requests")
	c.Abort()
}

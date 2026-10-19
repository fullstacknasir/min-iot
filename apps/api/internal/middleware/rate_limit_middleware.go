package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	RateLimitKeyPrefix = "ratelimit:login:%s"
	RateLimitTimeout   = 2 * time.Second
)

type memoryLimiter struct {
	sync.Mutex
	counts   map[string]int
	resetMap map[string]time.Time
}

var localLimiter = &memoryLimiter{
	counts:   make(map[string]int),
	resetMap: make(map[string]time.Time),
}

// LoginRateLimiter enforces maximum login attempts within a sliding time window.
func LoginRateLimiter(rdb *redis.Client, maxAttempts int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown_client"
		}

		allowed, remaining, err := checkRateLimit(c.Request.Context(), rdb, ip, maxAttempts, window)
		if err != nil {
			// Fail open on cache errors to avoid blocking authentications
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxAttempts))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many login attempts, please try again later",
				"retry_after_seconds": int(window.Seconds()),
			})
			return
		}

		c.Next()
	}
}

// APIRateLimiter enforces request rate limits across general endpoints.
func APIRateLimiter(rdb *redis.Client, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		if ip == "" {
			ip = "unknown_client"
		}

		key := fmt.Sprintf("api:%s", ip)
		allowed, remaining, err := checkRateLimit(c.Request.Context(), rdb, key, maxRequests, window)
		if err != nil {
			c.Next()
			return
		}

		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if !allowed {
			c.Header("Retry-After", fmt.Sprintf("%d", int(window.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":               "rate limit exceeded, please slow down your requests",
				"retry_after_seconds": int(window.Seconds()),
			})
			return
		}

		c.Next()
	}
}


func checkRateLimit(ctx context.Context, rdb *redis.Client, key string, maxAttempts int, window time.Duration) (bool, int, error) {
	if rdb != nil {
		return checkRedisRateLimit(ctx, rdb, key, maxAttempts, window)
	}
	return checkMemoryRateLimit(key, maxAttempts, window), 0, nil
}

func checkRedisRateLimit(ctx context.Context, rdb *redis.Client, key string, maxAttempts int, window time.Duration) (bool, int, error) {
	redisCtx, cancel := context.WithTimeout(ctx, RateLimitTimeout)
	defer cancel()

	redisKey := fmt.Sprintf(RateLimitKeyPrefix, key)
	count, err := rdb.Incr(redisCtx, redisKey).Result()
	if err != nil {
		return true, maxAttempts, err
	}

	if count == 1 {
		rdb.Expire(redisCtx, redisKey, window)
	}

	remaining := maxAttempts - int(count)
	if remaining < 0 {
		remaining = 0
	}

	return count <= int64(maxAttempts), remaining, nil
}

func checkMemoryRateLimit(key string, maxAttempts int, window time.Duration) bool {
	localLimiter.Lock()
	defer localLimiter.Unlock()

	now := time.Now()
	resetTime, exists := localLimiter.resetMap[key]
	if !exists || now.After(resetTime) {
		localLimiter.counts[key] = 1
		localLimiter.resetMap[key] = now.Add(window)
		return true
	}

	localLimiter.counts[key]++
	return localLimiter.counts[key] <= maxAttempts
}

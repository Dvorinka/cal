package httpapi

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// newRateLimiter limits each client IP to n requests per fixed window.
// In-memory only - per-instance, which is the right scope for a
// self-hosted single-container deployment.
func newRateLimiter(n int, window time.Duration) gin.HandlerFunc {
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := map[string]*bucket{}
	sweepAt := time.Now().Add(window)

	return func(c *gin.Context) {
		now := time.Now()
		ip := c.ClientIP()

		mu.Lock()
		if now.After(sweepAt) {
			for key, b := range buckets {
				if now.After(b.reset) {
					delete(buckets, key)
				}
			}
			sweepAt = now.Add(window)
		}
		b, ok := buckets[ip]
		if !ok || now.After(b.reset) {
			b = &bucket{reset: now.Add(window)}
			buckets[ip] = b
		}
		b.count++
		limited := b.count > n
		mu.Unlock()

		if limited {
			c.String(http.StatusTooManyRequests, "too many attempts, try again later")
			c.Abort()
			return
		}
		c.Next()
	}
}

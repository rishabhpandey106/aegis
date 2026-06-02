package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"github.com/aegis/firewall/internal/models"
	"github.com/aegis/firewall/internal/proxy"
)

type cacheEntry struct {
	route      *proxy.RouteConfig
	expiration time.Time
}

// InMemoryRouteCache implements proxy.ConfigProvider.
// It wraps database repositories with an in-memory TTL cache to prevent
// the proxy from querying the database on every single HTTP request.
type InMemoryRouteCache struct {
	projectRepo models.ProjectRepository
	ruleRepo    models.SecurityRuleRepository
	cache       sync.Map
	ttl         time.Duration
}

// NewInMemoryRouteCache initializes a new caching provider.
func NewInMemoryRouteCache(pRepo models.ProjectRepository, rRepo models.SecurityRuleRepository, ttl time.Duration) *InMemoryRouteCache {
	return &InMemoryRouteCache{
		projectRepo: pRepo,
		ruleRepo:    rRepo,
		ttl:         ttl,
	}
}

// GetRoute attempts to fetch the route from the cache using the raw API Key.
// It instantly hashes the key to protect it, and uses the hash for cache/DB lookups.
func (c *InMemoryRouteCache) GetRoute(ctx context.Context, apiKey string) (*proxy.RouteConfig, error) {
	// Hash the incoming raw API key
	hashBytes := sha256.Sum256([]byte(apiKey))
	hashStr := hex.EncodeToString(hashBytes[:])

	// 1. Check in-memory cache using the HASH as the key!
	if val, ok := c.cache.Load(hashStr); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiration) {
			return entry.route, nil
		}
		// Cache expired, remove it
		c.cache.Delete(hashStr)
	}

	// 2. Cache miss or expired, fetch project from DB using HASH
	project, err := c.projectRepo.GetByAPIKeyHash(ctx, hashStr)
	if err != nil {
		if err.Error() == "project not found" {
			return nil, errors.New("invalid api key")
		}
		return nil, err
	}

	// 3. Fetch security rules for project (still uses project.ID internally!)
	rules, err := c.ruleRepo.GetByProjectID(project.ID)
	if err != nil {
		return nil, err
	}

	ruleMap := make(map[string][]byte)
	for _, r := range rules {
		ruleMap[r.RuleType] = []byte(r.Configuration)
	}

	route := &proxy.RouteConfig{
		ProjectID:     project.ID,
		UpstreamURL:   project.UpstreamURL,
		IsActive:      project.IsActive,
		SecurityRules: ruleMap,
	}

	// 4. Save to cache with TTL
	c.cache.Store(hashStr, cacheEntry{
		route:      route,
		expiration: time.Now().Add(c.ttl),
	})

	return route, nil
}

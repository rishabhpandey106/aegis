package cache

import (
	"context"
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
// It wraps a database repository with an in-memory TTL cache to prevent 
// the proxy from querying the database on every single HTTP request.
type InMemoryRouteCache struct {
	repo  models.ProjectRepository
	cache sync.Map
	ttl   time.Duration
}

// NewInMemoryRouteCache initializes a new caching provider.
func NewInMemoryRouteCache(repo models.ProjectRepository, ttl time.Duration) *InMemoryRouteCache {
	return &InMemoryRouteCache{
		repo: repo,
		ttl:  ttl,
	}
}

// GetRoute attempts to fetch the route from the cache. 
// If it is missing or expired, it queries the underlying database.
func (c *InMemoryRouteCache) GetRoute(ctx context.Context, projectID string) (*proxy.RouteConfig, error) {
	// 1. Check in-memory cache
	if val, ok := c.cache.Load(projectID); ok {
		entry := val.(cacheEntry)
		if time.Now().Before(entry.expiration) {
			return entry.route, nil
		}
		// Cache expired, remove it
		c.cache.Delete(projectID)
	}

	// 2. Cache miss or expired, fetch from DB
	project, err := c.repo.GetByID(ctx, projectID)
	if err != nil {
		if err.Error() == "project not found" {
			return nil, errors.New("project not found")
		}
		return nil, err
	}

	route := &proxy.RouteConfig{
		ProjectID:   project.ID,
		UpstreamURL: project.UpstreamURL,
		IsActive:    project.IsActive,
	}

	// 3. Save to cache with TTL
	c.cache.Store(projectID, cacheEntry{
		route:      route,
		expiration: time.Now().Add(c.ttl),
	})

	return route, nil
}

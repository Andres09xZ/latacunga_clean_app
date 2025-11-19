package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/models"
	"gorm.io/gorm"
)

// DistanceMatrixCache manages caching of distance matrices
type DistanceMatrixCache struct {
	db         *gorm.DB
	ttl        time.Duration
	memCache   map[string]*CachedMatrix
	cacheMutex sync.RWMutex
}

// CachedMatrix holds cached matrix data
type CachedMatrix struct {
	Data      json.RawMessage
	ExpiresAt time.Time
}

// NewDistanceMatrixCache creates a new cache manager
func NewDistanceMatrixCache(db *gorm.DB, ttlMinutes int) *DistanceMatrixCache {
	return &DistanceMatrixCache{
		db:       db,
		ttl:      time.Duration(ttlMinutes) * time.Minute,
		memCache: make(map[string]*CachedMatrix),
	}
}

// Get retrieves a cached distance matrix
func (c *DistanceMatrixCache) Get(pointsHash string) (json.RawMessage, error) {
	// Check in-memory cache first
	c.cacheMutex.RLock()
	if cached, exists := c.memCache[pointsHash]; exists {
		c.cacheMutex.RUnlock()
		if time.Now().Before(cached.ExpiresAt) {
			log.Printf("✅ Cache HIT for points hash: %s", pointsHash)
			return cached.Data, nil
		}
		// Expired, remove from cache
		c.cacheMutex.Lock()
		delete(c.memCache, pointsHash)
		c.cacheMutex.Unlock()
	}
	c.cacheMutex.RUnlock()

	// Check database
	var cacheEntry models.DistanceMatrixCache
	if err := c.db.Where("key_hash = ? AND ttl_until > ?", pointsHash, time.Now()).
		First(&cacheEntry).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // Cache miss
		}
		log.Printf("⚠️  Error reading cache: %v", err)
		return nil, err
	}

	// Store in memory cache
	c.cacheMutex.Lock()
	c.memCache[pointsHash] = &CachedMatrix{
		Data:      cacheEntry.Payload,
		ExpiresAt: cacheEntry.TTLUntil,
	}
	c.cacheMutex.Unlock()

	log.Printf("✅ Cache HIT (from DB) for points hash: %s", pointsHash)
	return cacheEntry.Payload, nil
}

// Set stores a distance matrix in cache
func (c *DistanceMatrixCache) Set(pointsHash string, data json.RawMessage) error {
	ttlUntil := time.Now().Add(c.ttl)

	// Store in database
	cacheEntry := models.DistanceMatrixCache{
		KeyHash:  pointsHash,
		Payload:  data,
		TTLUntil: ttlUntil,
	}

	if err := c.db.Create(&cacheEntry).Error; err != nil {
		log.Printf("⚠️  Error storing cache: %v", err)
	}

	// Store in memory cache
	c.cacheMutex.Lock()
	c.memCache[pointsHash] = &CachedMatrix{
		Data:      data,
		ExpiresAt: ttlUntil,
	}
	c.cacheMutex.Unlock()

	log.Printf("✅ Cache SET for points hash: %s (TTL: %v)", pointsHash, c.ttl)
	return nil
}

// GeneratePointsHash creates a unique hash for a set of points
func GeneratePointsHash(locations []string) string {
	data := ""
	for _, loc := range locations {
		data += loc + "|"
	}
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// CleanupExpired removes expired entries from cache
func (c *DistanceMatrixCache) CleanupExpired() error {
	result := c.db.Where("ttl_until < ?", time.Now()).Delete(&models.DistanceMatrixCache{})
	if result.Error != nil {
		return fmt.Errorf("failed to cleanup expired cache: %w", result.Error)
	}
	log.Printf("✅ Cleaned up %d expired cache entries", result.RowsAffected)
	return nil
}

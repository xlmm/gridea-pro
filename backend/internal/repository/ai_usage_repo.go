package repository

import (
	"context"
	"gridea-pro/backend/internal/domain"
	"path/filepath"
	"sync"
)

type aiUsageRepository struct {
	mu     sync.RWMutex
	appDir string
	cache  *domain.AIUsage
	loaded bool
}

func NewAIUsageRepository(appDir string) domain.AIUsageRepository {
	return &aiUsageRepository{appDir: appDir}
}

func (r *aiUsageRepository) filePath() string {
	return filepath.Join(r.appDir, "config", "ai_usage.json")
}

func (r *aiUsageRepository) loadIfNeeded() {
	r.mu.RLock()
	if r.loaded {
		r.mu.RUnlock()
		return
	}
	r.mu.RUnlock()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.loaded {
		return
	}

	var usage domain.AIUsage
	if err := LoadJSONFile(r.filePath(), &usage); err != nil {
		r.cache = &domain.AIUsage{}
	} else {
		r.cache = &usage
	}
	r.loaded = true
}

func (r *aiUsageRepository) GetAIUsage(ctx context.Context) (domain.AIUsage, error) {
	r.loadIfNeeded()

	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.cache == nil {
		return domain.AIUsage{}, nil
	}
	return *r.cache, nil
}

func (r *aiUsageRepository) SaveAIUsage(ctx context.Context, usage domain.AIUsage) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := SaveJSONFile(r.filePath(), usage); err != nil {
		return err
	}
	r.cache = &usage
	r.loaded = true
	return nil
}

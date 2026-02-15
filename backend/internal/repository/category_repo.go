package repository

import (
	"context"
	"gridea-pro/backend/internal/domain"
)

type categoryRepository struct {
	*BaseJSONRepository[domain.Category]
}

func NewCategoryRepository(appDir string) domain.CategoryRepository {
	base := NewBaseJSONRepository[domain.Category](appDir, "categories.json", "categories")
	return &categoryRepository{base}
}

func (r *categoryRepository) Create(ctx context.Context, category *domain.Category) error {
	return r.Add(ctx, *category)
}

func (r *categoryRepository) Update(ctx context.Context, slug string, category *domain.Category) error {
	return r.BaseJSONRepository.Update(ctx, slug, *category)
}

func (r *categoryRepository) GetBySlug(ctx context.Context, slug string) (*domain.Category, error) {
	cat, err := r.Get(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &cat, nil
}

// List overrides generic list if needed, or uses it directly.
// But interface match: List(ctx) ([]Category, error)
// Generic: List(ctx) ([]T, error) -> T is Category -> []Category. Matches.

// SaveAll overrides generic SaveAll if needed.
// Generic: SaveAll(ctx, []T) error. Matches.

package repository

import (
	"context"
	"gridea-pro/backend/internal/domain"
)

type tagRepository struct {
	*BaseJSONRepository[domain.Tag]
}

func NewTagRepository(appDir string) domain.TagRepository {
	base := NewBaseJSONRepository[domain.Tag](appDir, "tags.json", "tags")
	return &tagRepository{base}
}

func (r *tagRepository) Create(ctx context.Context, tag *domain.Tag) error {
	return r.Add(ctx, *tag)
}

func (r *tagRepository) Update(ctx context.Context, tag *domain.Tag) error {
	return r.BaseJSONRepository.Update(ctx, tag.ID, *tag)
}

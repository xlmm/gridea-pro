package repository

import (
	"context"
	"gridea-pro/backend/internal/domain"
)

type linkRepository struct {
	*BaseJSONRepository[domain.Link]
}

func NewLinkRepository(appDir string) domain.LinkRepository {
	base := NewBaseJSONRepository[domain.Link](appDir, "links.json", "links")
	return &linkRepository{base}
}

func (r *linkRepository) Create(ctx context.Context, link *domain.Link) error {
	return r.Add(ctx, *link)
}

func (r *linkRepository) Update(ctx context.Context, id string, link *domain.Link) error {
	return r.BaseJSONRepository.Update(ctx, id, *link)
}

func (r *linkRepository) GetByID(ctx context.Context, id string) (*domain.Link, error) {
	link, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &link, nil
}

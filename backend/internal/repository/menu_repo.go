package repository

import (
	"context"
	"gridea-pro/backend/internal/domain"
)

type menuRepository struct {
	*BaseJSONRepository[domain.Menu]
}

func NewMenuRepository(appDir string) domain.MenuRepository {
	base := NewBaseJSONRepository[domain.Menu](appDir, "menus.json", "menus")
	return &menuRepository{base}
}

func (r *menuRepository) Create(ctx context.Context, menu *domain.Menu) error {
	return r.Add(ctx, *menu)
}

func (r *menuRepository) Update(ctx context.Context, id string, menu *domain.Menu) error {
	return r.BaseJSONRepository.Update(ctx, id, *menu)
}

func (r *menuRepository) GetByID(ctx context.Context, id string) (*domain.Menu, error) {
	menu, err := r.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return &menu, nil
}

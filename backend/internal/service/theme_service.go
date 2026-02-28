package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"gridea-pro/backend/internal/domain"
)

type ThemeService struct {
	repo   domain.ThemeRepository
	appDir string
	mu     sync.RWMutex
}

func NewThemeService(repo domain.ThemeRepository, appDir string) *ThemeService {
	return &ThemeService{repo: repo, appDir: appDir}
}

func (s *ThemeService) LoadThemes(ctx context.Context) ([]domain.Theme, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.repo.GetAll(ctx)
}

func (s *ThemeService) LoadThemeConfig(ctx context.Context) (domain.ThemeConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.repo.GetConfig(ctx)
}

func (s *ThemeService) SaveThemeConfig(ctx context.Context, config domain.ThemeConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Should update config.json
	return s.repo.SaveConfig(ctx, config)
}

func (s *ThemeService) SaveThemeImage(ctx context.Context, sourcePath string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceFileStat, err := os.Stat(sourcePath)
	if err != nil {
		return "", err
	}
	if !sourceFileStat.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", sourcePath)
	}

	fileName := filepath.Base(sourcePath)
	destRelativePath := filepath.Join("images", "theme", fileName)
	destPath := filepath.Join(s.appDir, destRelativePath)

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return "", err
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return "", err
	}
	defer source.Close()

	destination, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return "", err
	}

	return "/" + filepath.ToSlash(destRelativePath), nil
}

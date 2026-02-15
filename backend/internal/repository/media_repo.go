package repository

import (
	"context"
	"fmt"
	"gridea-pro/backend/internal/domain"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type mediaRepository struct {
	mu     sync.RWMutex
	appDir string
}

func NewMediaRepository(appDir string) domain.MediaRepository {
	return &mediaRepository{appDir: appDir}
}

func (r *mediaRepository) SaveImages(ctx context.Context, files []domain.UploadedFile) ([]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	postImageDir := filepath.Join(r.appDir, "post-images")
	_ = os.MkdirAll(postImageDir, 0755)

	var results []string
	for i, file := range files {
		ext := filepath.Ext(file.Name)
		// Use UnixNano and index to ensure uniqueness even in same batch
		newName := fmt.Sprintf("%d_%d%s", time.Now().UnixNano(), i, ext)
		newPath := filepath.Join(postImageDir, newName)

		if err := CopyFile(file.Path, newPath); err != nil {
			continue
		}
		// Return relative path for frontend usage
		results = append(results, "/post-images/"+newName)
	}

	return results, nil
}

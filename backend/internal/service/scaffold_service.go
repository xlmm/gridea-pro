package service

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

type ScaffoldService struct {
	assets embed.FS
	mu     sync.Mutex
}

func NewScaffoldService(assets embed.FS) *ScaffoldService {
	return &ScaffoldService{
		assets: assets,
	}
}

// InitSite checks if the site is initialized, if not, it copies default files.
// A .initialized marker file is used to track whether the site has been scaffolded.
// Default content (posts, memos, etc.) is only copied on first initialization.
// Essential directories and config patches are applied on every startup.
func (s *ScaffoldService) InitSite(appDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	markerPath := filepath.Join(appDir, ".initialized")
	isFirstInit := false

	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		isFirstInit = true
	}

	if isFirstInit {
		// 1. Locate default-files source path in embed.FS
		srcParams := []string{"frontend/dist/default-files", "frontend/public/default-files"}
		var srcPath string

		for _, p := range srcParams {
			if _, err := fs.Stat(s.assets, p); err == nil {
				srcPath = p
				break
			}
		}

		if srcPath == "" {
			return fmt.Errorf("default-files not found in assets")
		}

		// 2. Recursively copy all default files to appDir
		if err := s.copyDirFromEmbed(srcPath, appDir); err != nil {
			return fmt.Errorf("failed to copy default files: %w", err)
		}

		// 3. Fill date placeholders in default posts and memos with current time
		s.fillDefaultDates(appDir)

		// 4. Create marker file
		_ = os.WriteFile(markerPath, []byte("Gridea Pro initialized\n"), 0644)
	}

	// Always ensure essential directories exist
	_ = os.MkdirAll(filepath.Join(appDir, "output"), 0755)

	// Always patch config.json with current sourceFolder
	configPath := filepath.Join(appDir, "config", "config.json")
	if content, err := os.ReadFile(configPath); err == nil {
		var config map[string]interface{}
		if err := json.Unmarshal(content, &config); err == nil {
			config["sourceFolder"] = appDir
			if data, err := json.MarshalIndent(config, "", "  "); err == nil {
				_ = os.WriteFile(configPath, data, 0644)
			}
		}
	}

	return nil
}

// fillDefaultDates replaces date placeholders in default posts and memos with current time.
//
// Posts use placeholders like __INIT_DATE_00__, __INIT_DATE_01__, etc. in frontmatter.
// The number suffix controls sort order: 00 is newest, higher numbers are older.
// Each increment adds a 1-minute offset into the past.
//
// Memos use empty createdAt/updatedAt strings ("") which are replaced with current time.
func (s *ScaffoldService) fillDefaultDates(appDir string) {
	now := time.Now()

	// Fill post dates — scan all .md files for __INIT_DATE_XX__ placeholders
	postsDir := filepath.Join(appDir, "posts")
	entries, err := os.ReadDir(postsDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		postPath := filepath.Join(postsDir, entry.Name())
		data, err := os.ReadFile(postPath)
		if err != nil {
			continue
		}
		// Match __INIT_DATE_XX__ where XX is 00-99
		if !bytes.Contains(data, []byte("__INIT_DATE_")) {
			continue
		}
		newData := data
		for i := 0; i < 100; i++ {
			placeholder := fmt.Sprintf("__INIT_DATE_%02d__", i)
			if bytes.Contains(newData, []byte(placeholder)) {
				postTime := now.Add(-time.Duration(i) * time.Minute)
				dateStr := postTime.Format("2006-01-02 15:04:05")
				newData = bytes.Replace(newData, []byte(placeholder), []byte(dateStr), 1)
				break
			}
		}
		if !bytes.Equal(data, newData) {
			_ = os.WriteFile(postPath, newData, 0644)
		}
	}

	// Fill memo dates — replace empty strings with current time
	memosPath := filepath.Join(appDir, "config", "memos.json")
	memosData, err := os.ReadFile(memosPath)
	if err != nil {
		return
	}
	if bytes.Contains(memosData, []byte(`"createdAt": ""`)) {
		memoTime := now.Format(time.RFC3339)
		memosData = bytes.ReplaceAll(memosData, []byte(`"createdAt": ""`), []byte(`"createdAt": "`+memoTime+`"`))
		memosData = bytes.ReplaceAll(memosData, []byte(`"updatedAt": ""`), []byte(`"updatedAt": "`+memoTime+`"`))
		_ = os.WriteFile(memosPath, memosData, 0644)
	}
}

func (s *ScaffoldService) copyDirFromEmbed(src string, dst string) error {
	entries, err := s.assets.ReadDir(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	for _, entry := range entries {
		// embed.FS always uses forward slashes, even on Windows
		srcPath := path.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := s.copyDirFromEmbed(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := s.copyFileFromEmbed(srcPath, dstPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ScaffoldService) copyFileFromEmbed(src string, dst string) error {
	// Check if destination file exists
	if _, err := os.Stat(dst); err == nil {
		// File exists, skip
		return nil
	}

	sourceFile, err := s.assets.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

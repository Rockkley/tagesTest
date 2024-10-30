package storage

import (
	"io"
	"os"
	"path/filepath"
	"sync"
	"tagesTest/internal/domain"
)

type DiskStorage struct {
	baseDir string
	mu      sync.RWMutex
}

func NewDiskStorage(baseDir string) *DiskStorage {
	return &DiskStorage{baseDir: baseDir}
}

func (s *DiskStorage) Save(filename string, reader io.Reader) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filePath := filepath.Join(s.baseDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, reader)
	return err
}

func (s *DiskStorage) List() ([]domain.File, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var files []domain.File
	err := filepath.Walk(s.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, domain.File{
				Filename:  info.Name(),
				CreatedAt: info.ModTime(),
				UpdatedAt: info.ModTime(),
			})
		}
		return nil
	})
	return files, err
}

func (s *DiskStorage) Get(filename string) (io.ReadCloser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return os.Open(filename)
}

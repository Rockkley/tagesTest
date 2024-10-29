package repository

import (
	"io"
	"tagesTest/internal/domain"
	"tagesTest/internal/storage"
)

type FileRepository struct {
	storage storage.FileStorageInterface
}

func NewFileRepository(storage storage.FileStorageInterface) *FileRepository {
	return &FileRepository{storage: storage}
}

func (r *FileRepository) SaveFile(filename string, reader io.Reader) error {
	return r.storage.Save(filename, reader)
}

func (r *FileRepository) ListFiles() ([]domain.File, error) {
	return r.storage.List()
}

func (r *FileRepository) GetFile(filename string) (io.ReadCloser, error) {
	return r.storage.Get(filename)
}

package storage

import (
	"io"
	"tagesTest/internal/domain"
)

type FileStorageInterface interface {
	Save(filename string, reader io.Reader) error
	List() ([]domain.File, error)
	Get(filename string) (io.ReadCloser, error)
}

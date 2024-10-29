package service

import (
	"io"
	"tagesTest/internal/domain"
	"tagesTest/internal/repository"
)

type FileService struct {
	repo *repository.FileRepository
}

func NewFileService(repo *repository.FileRepository) *FileService {
	return &FileService{repo: repo}
}

func (s *FileService) SaveFile(filename string, reader io.Reader) error {
	return s.repo.SaveFile(filename, reader)
}

func (s *FileService) ListFiles() ([]domain.File, error) {
	return s.repo.ListFiles()
}

func (s *FileService) DownloadFile(filename string) (io.ReadCloser, error) {
	return s.repo.GetFile(filename)
}

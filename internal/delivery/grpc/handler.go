package grpc

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"tagesTest/internal/service"
	"tagesTest/pkg/limiter"
	pb "tagesTest/proto"
)

const uploadBufferSize = 100

type FileServiceHandler struct {
	pb.UnimplementedFileServiceServer
	service          *service.FileService
	uploadLimiter    *limiter.Limiter
	downloadLimiter  *limiter.Limiter
	listLimiter      *limiter.Limiter
	storageDir       string
	uploadBufferSize int
}

func NewFileServiceHandler(service *service.FileService, storageDir string) *FileServiceHandler {
	return &FileServiceHandler{
		service:          service,
		uploadLimiter:    limiter.NewLimiter(10),
		downloadLimiter:  limiter.NewLimiter(10),
		listLimiter:      limiter.NewLimiter(100),
		storageDir:       storageDir,
		uploadBufferSize: uploadBufferSize,
	}
}

func (h *FileServiceHandler) UploadFile(stream pb.FileService_UploadFileServer) error {
	log.Println("Starting file upload process")

	ctx, cancel := context.WithTimeout(stream.Context(), 30*time.Second)
	defer cancel()

	if err := h.uploadLimiter.Acquire(ctx); err != nil {
		log.Println("uploaders limit reached")
		return status.Errorf(codes.ResourceExhausted, "uploaders limit reached")
	}
	defer h.uploadLimiter.Release()

	var filename string
	var totalSize int64
	dataChan := make(chan []byte, h.uploadBufferSize)
	errChan := make(chan error, 1)

	go func() {
		defer close(dataChan)
		for {
			select {
			case <-ctx.Done():
				log.Println("Context done, stopping receive loop")
				return
			default:
				req, err := stream.Recv()

				if err == io.EOF {
					log.Println("received EOF in goroutine")
					return
				}
				if err != nil {
					log.Printf("error receiving chunk in goroutine: %v", err)
					errChan <- err
					return
				}
				if filename == "" {
					filename = filepath.Base(req.Filename)
					log.Printf("Received filename: %s", filename)
				}
				dataChan <- req.Chunk
			}
		}
	}()

	filePath := filepath.Join(h.storageDir, filename)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		log.Printf("failed to create directory: %v", err)
		return status.Errorf(codes.Internal, "failed to create directory: %v", err)
	}

	// check if file already exists
	if _, err := os.Stat(filePath); err == nil {
		return status.Errorf(codes.AlreadyExists, "file already exists")
	}

	file, err := os.Create(filePath)
	if err != nil {
		log.Printf("failed to create file: %v", err)
		return status.Errorf(codes.Internal, "failed to create file: %v", err)
	}
	defer file.Close()

	for {
		select {
		case chunk, ok := <-dataChan:
			if !ok {
				if totalSize == 0 {
					log.Println("no file data received")
					return status.Errorf(codes.InvalidArgument, "no file data received")
				}
				log.Printf("File %s uploaded successfully, size: %d bytes", filename, totalSize)
				return stream.SendAndClose(&pb.UploadFileResponse{Message: "File uploaded successfully"})
			}
			_, err := file.Write(chunk)
			if err != nil {
				log.Printf("failed to write chunk: %v", err)
				return status.Errorf(codes.Internal, "failed to write chunk: %v", err)
			}
			totalSize += int64(len(chunk))
			log.Printf("Received chunk, total data size: %d bytes", totalSize)
		case err := <-errChan:
			return status.Errorf(codes.Internal, "error receiving file: %v", err)
		}
	}
}

func (h *FileServiceHandler) ListFiles(ctx context.Context, req *pb.ListFilesRequest) (*pb.ListFilesResponse, error) {
	if err := h.listLimiter.Acquire(ctx); err != nil {
		return nil, status.Errorf(codes.ResourceExhausted, "list limit reached")
	}
	defer h.listLimiter.Release()

	files, err := h.service.ListFiles()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to list files: %v", err)
	}

	var pbFiles []*pb.FileInfo
	for _, file := range files {
		pbFiles = append(pbFiles, &pb.FileInfo{
			Filename:  file.Filename,
			CreatedAt: file.CreatedAt.Format(time.RFC3339),
			UpdatedAt: file.UpdatedAt.Format(time.RFC3339),
		})
	}

	return &pb.ListFilesResponse{Files: pbFiles}, nil
}

func (h *FileServiceHandler) DownloadFile(req *pb.DownloadFileRequest, stream pb.FileService_DownloadFileServer) error {
	if err := h.downloadLimiter.Acquire(stream.Context()); err != nil {
		return status.Errorf(codes.ResourceExhausted, "download limit reached")
	}
	defer h.downloadLimiter.Release()
	fmt.Println(h.storageDir, req.Filename)
	filePath := filepath.Join(h.storageDir, req.Filename)
	file, err := h.service.DownloadFile(filePath)
	if err != nil {
		return status.Errorf(codes.NotFound, "file not found: %v", err)
	}
	defer file.Close()

	buffer := make([]byte, 1024)
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to read file: %v", err)
		}
		if err := stream.Send(&pb.DownloadFileResponse{Chunk: buffer[:n]}); err != nil {
			return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
		}
	}

	return nil
}

func isImage(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp":
		return true
	default:
		return false
	}
}
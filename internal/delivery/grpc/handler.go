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
	filename         string
	uploadBufferSize int
	errChan          chan error
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

	if err := os.MkdirAll(filepath.Dir(h.storageDir), 0755); err != nil {
		log.Printf("failed to create directory: %v", err)
		return status.Errorf(codes.Internal, "failed to create directory: %v", err)
	}
	var (
		filename  string
		totalSize int64
	)

	dataChan := make(chan []byte, h.uploadBufferSize)
	errChan := make(chan error, 1)

	req, err := stream.Recv()
	if err == io.EOF {
		log.Println("received EOF in goroutine")
	}
	h.filename = filepath.Base(req.GetImagePath())
	filePath := fmt.Sprintf("%s/%s", h.storageDir, h.filename)

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

	go h.receivingLoop(ctx, dataChan, stream)
	for {
		select {
		case chunk, ok := <-dataChan:
			if !ok {
				if totalSize == 0 {
					log.Println("No file data received")
					return status.Error(codes.InvalidArgument, "No file data received")
				}
				log.Printf("File %s uploaded successfully, size: %d bytes", filename, totalSize)
				return stream.SendAndClose(&pb.UploadFileResponse{
					Message: fmt.Sprintf("File uploaded successfully. Size: %d bytes", totalSize),
				})
			}
			n, err := file.Write(chunk)
			if err != nil {
				log.Printf("Failed to write chunk: %v", err)
				return status.Errorf(codes.Internal, "Failed to write chunk: %v", err)
			}
			totalSize += int64(n)
			log.Printf("Received chunk, total data size: %d bytes", totalSize)
		case err := <-errChan:
			return status.Errorf(codes.Internal, "Error receiving file: %v", err)
		}
	}
}

func (h *FileServiceHandler) receivingLoop(
	ctx context.Context, dataChan chan []byte, stream pb.FileService_UploadFileServer,
) {

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
				h.errChan <- err
				return
			}
			chunk := req.GetChunk()
			size := len(chunk)
			log.Printf("received a chunk with size: %d", size)
			dataChan <- chunk
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

	if !isImage(req.Filename) {
		return status.Errorf(codes.InvalidArgument, "not an image")
	}

	sourcePath := filepath.Join(h.storageDir, req.Filename)
	sourceFile, err := h.service.DownloadFile(sourcePath)
	if err != nil {
		return status.Errorf(codes.NotFound, "file not found: %v", err)
	}
	defer sourceFile.Close()

	destPath := filepath.Join("downloads", "downloaded_"+req.Filename)
	destFile, err := os.Create(destPath)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create destination file: %v", err)
	}
	defer destFile.Close()

	buffer := make([]byte, 1024)
	for {
		n, err := sourceFile.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to read file: %v", err)
		}

		_, err = destFile.Write(buffer[:n])
		if err != nil {
			return status.Errorf(codes.Internal, "failed to write to destination file: %v", err)
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

package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/grpc"
	pb "tagesTest/proto"
)

const downloadDir = "./downloads"

func main() {
	conn, err := grpc.NewClient("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("failed to connect: %v", err)
	}
	defer conn.Close()

	client := pb.NewFileServiceClient(conn)

	uploadFile(client, "./files/ruru.bmp")
	downloadFile(client, "ruru.bmp")

	listFiles(client)
}

func uploadFile(client pb.FileServiceClient, filePath string) {
	if !isImage(filePath) {
		fmt.Println("File is not an image.")
		return
	}
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.UploadFile(ctx)
	if err != nil {
		log.Fatal("cannot upload image: ", err)
	}

	req := &pb.UploadFileRequest{
		Data: &pb.UploadFileRequest_ImagePath{
			ImagePath: filePath,
		},
	}

	err = stream.Send(req)
	if err != nil {
		log.Fatal("cannot send image info to server: ", err, stream.RecvMsg(nil))
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal("cannot read chunk to buffer: ", err)
		}

		req := &pb.UploadFileRequest{
			Data: &pb.UploadFileRequest_Chunk{
				Chunk: buffer[:n],
			},
		}

		err = stream.Send(req)
		if err != nil {
			log.Fatal("cannot send chunk to server: ", err)
		}
	}
	res, err := stream.CloseAndRecv()
	if err != nil {
		log.Fatal("cannot receive response: ", err)
	}

	log.Printf("image uploaded, size: %d", res.GetSize())
}

func downloadFile(client pb.FileServiceClient, filePath string) {
	if !isImage(filePath) {
		fmt.Println("File is not an image.")
		return
	}
	req := &pb.DownloadFileRequest{Filename: filePath}
	stream, err := client.DownloadFile(context.Background(), req)
	if err != nil {
		log.Fatalf("error downloading file: %v", err)
	}
	err = os.MkdirAll(downloadDir, 0755)
	if err != nil {
		return
	}
	file, err := os.Create(filepath.Join(downloadDir, filePath))
	if err != nil {
		log.Fatalf("error creating file: %v", err)
	}
	defer file.Close()

	for {
		resp, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("error receiving chunk: %v", err)
		}
		_, err = file.Write(resp.Chunk)
		if err != nil {
			log.Fatalf("error writing chunk: %v", err)
		}
	}

	fmt.Printf("file %s downloaded successfully\n", filePath)
}

func listFiles(client pb.FileServiceClient) {
	resp, err := client.ListFiles(context.Background(), &pb.ListFilesRequest{})
	if err != nil {
		log.Fatalf("error listing files: %v", err)
	}

	fmt.Println("files:")
	for _, file := range resp.Files {
		fmt.Printf("- %s (Created: %s, Updated: %s)\n", file.Filename, file.CreatedAt, file.UpdatedAt)
	}
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

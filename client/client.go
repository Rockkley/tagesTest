package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

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

	downloadFile(client, "test.txt")

	listFiles(client)
}

func downloadFile(client pb.FileServiceClient, filename string) {
	req := &pb.DownloadFileRequest{Filename: filename}
	stream, err := client.DownloadFile(context.Background(), req)
	if err != nil {
		log.Fatalf("error downloading file: %v", err)
	}
	err = os.MkdirAll(downloadDir, 0755)
	if err != nil {
		return
	}
	file, err := os.Create(filepath.Join(downloadDir, filename))
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

	fmt.Printf("file %s downloaded successfully\n", filename)
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

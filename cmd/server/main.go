package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"tagesTest/internal/config"
	"tagesTest/internal/delivery/grpc"
	"tagesTest/internal/repository"
	"tagesTest/internal/service"
	"tagesTest/internal/storage"
)

func main() {
	cfg := config.Load()

	fileStorage := storage.NewDiskStorage(cfg.StorageDir)
	fileRepo := repository.NewFileRepository(fileStorage)
	fileService := service.NewFileService(fileRepo)

	server, err := grpc.NewServer(cfg.ServerAddress, fileService, cfg.StorageDir)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	go func() {
		log.Printf("Starting gRPC server on %s", cfg.ServerAddress)
		if err := server.Start(); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	server.Stop()
	log.Println("Server stopped")
}

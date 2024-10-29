package grpc

import (
	"net"

	"google.golang.org/grpc"
	"tagesTest/internal/service"
	pb "tagesTest/proto"
)

type Server struct {
	listener net.Listener
	server   *grpc.Server
}

func NewServer(address string, fileService *service.FileService, storageDir string) (*Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	server := grpc.NewServer()
	handler := NewFileServiceHandler(fileService, storageDir)
	pb.RegisterFileServiceServer(server, handler)

	return &Server{
		listener: listener,
		server:   server,
	}, nil
}

func (s *Server) Start() error {
	return s.server.Serve(s.listener)
}

func (s *Server) Stop() {
	s.server.GracefulStop()
}

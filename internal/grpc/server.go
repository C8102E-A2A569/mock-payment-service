package grpcserver

import (
	"net"

	pb "new-project/api/proto/payment"
	"new-project/internal/grpc/handlers"
	"new-project/internal/service"

	"google.golang.org/grpc"
)

// NewServer создаёт gRPC-сервер и регистрирует сервис платежей.
func NewServer(paymentSvc *service.PaymentService) *grpc.Server {
	s := grpc.NewServer()
	h := handlers.NewPaymentHandler(paymentSvc)
	pb.RegisterPaymentServiceServer(s, h)
	return s
}

// Listen создаёт TCP-listener на указанном адресе (например, ":50051").
func Listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}


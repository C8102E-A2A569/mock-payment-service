package grpcserver

import (
	"net"

	pb "new-project/api/proto/payment"
	"new-project/internal/grpc/handlers"
	"new-project/internal/service"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func NewServer(paymentSvc *service.PaymentService) *grpc.Server {
	s := grpc.NewServer()
	h := handlers.NewPaymentHandler(paymentSvc)
	pb.RegisterPaymentServiceServer(s, h)
	reflection.Register(s)
	return s
}

func Listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

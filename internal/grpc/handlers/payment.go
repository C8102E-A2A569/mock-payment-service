package handlers

import (
	"context"
	"errors"

	pb "new-project/api/proto/payment"
	"new-project/internal/service"
	"new-project/pkg/apperror"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// PaymentHandler реализует gRPC-интерфейс PaymentService.
type PaymentHandler struct {
	pb.UnimplementedPaymentServiceServer
	svc *service.PaymentService
}

// NewPaymentHandler создаёт обработчик запросов PaymentService.
func NewPaymentHandler(svc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{svc: svc}
}

func (h *PaymentHandler) CreateAccount(ctx context.Context, req *pb.CreateAccountRequest) (*pb.CreateAccountResponse, error) {
	if req.GetUserId() == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	acc, err := h.svc.CreateAccount(ctx, req.GetUserId())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &pb.CreateAccountResponse{
		AccountId: acc.ID.String(),
		Balance:   acc.Balance,
	}, nil
}

func (h *PaymentHandler) GetBalance(ctx context.Context, req *pb.GetBalanceRequest) (*pb.GetBalanceResponse, error) {
	accountID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}
	balance, err := h.svc.GetBalance(ctx, accountID)
	if err != nil {
		return nil, toStatusError(err)
	}
	return &pb.GetBalanceResponse{
		AccountId: req.GetAccountId(),
		Balance:   balance,
	}, nil
}

func (h *PaymentHandler) Deposit(ctx context.Context, req *pb.DepositRequest) (*pb.DepositResponse, error) {
	accountID, err := uuid.Parse(req.GetAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}
	txID, newBalance, err := h.svc.Deposit(ctx, accountID, req.GetAmount(), req.GetIdempotencyKey())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &pb.DepositResponse{
		TransactionId: txID.String(),
		NewBalance:    newBalance,
	}, nil
}

func (h *PaymentHandler) Transfer(ctx context.Context, req *pb.TransferRequest) (*pb.TransferResponse, error) {
	fromID, err := uuid.Parse(req.GetFromAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid from_account_id")
	}
	toID, err := uuid.Parse(req.GetToAccountId())
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid to_account_id")
	}
	txID, fromBalance, toBalance, err := h.svc.Transfer(ctx, fromID, toID, req.GetAmount(), req.GetIdempotencyKey())
	if err != nil {
		return nil, toStatusError(err)
	}
	return &pb.TransferResponse{
		TransactionId: txID.String(),
		FromNewBalance: fromBalance,
		ToNewBalance:   toBalance,
		Success:        true,
	}, nil
}

// toStatusError маппит AppError на gRPC status.Error с корректным codes.Code.
func toStatusError(err error) error {
	if err == nil {
		return nil
	}
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		var code codes.Code
		switch appErr.Code {
		case apperror.CodeInvalidArgument:
			code = codes.InvalidArgument
		case apperror.CodeNotFound:
			code = codes.NotFound
		case apperror.CodeInsufficientFunds:
			code = codes.FailedPrecondition
		default:
			code = codes.Internal
		}
		return status.Error(code, appErr.Msg)
	}
	return status.Error(codes.Internal, "internal error")
}


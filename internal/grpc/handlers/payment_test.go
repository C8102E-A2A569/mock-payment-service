package handlers

import (
	"context"
	"errors"
	"testing"

	pb "new-project/api/proto/payment"
	"new-project/internal/service"
	"new-project/internal/testutil"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPaymentHandler_CreateAccount_ValidationAndSuccess(t *testing.T) {
	repo := testutil.NewFakeRepo()
	svc := service.NewPaymentService(repo, nil, nil)
	h := NewPaymentHandler(svc)
	ctx := context.Background()

	// user_id обязателен
	_, err := h.CreateAccount(ctx, &pb.CreateAccountRequest{})
	if err == nil {
		t.Fatal("expected error when user_id is empty")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}

	// успешное создание
	resp, err := h.CreateAccount(ctx, &pb.CreateAccountRequest{UserId: "user-1"})
	if err != nil {
		t.Fatalf("CreateAccount: %v", err)
	}
	if resp.GetAccountId() == "" || resp.GetBalance() != 0 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestPaymentHandler_GetBalance_InvalidUUIDAndNotFound(t *testing.T) {
	repo := testutil.NewFakeRepo()
	svc := service.NewPaymentService(repo, nil, nil)
	h := NewPaymentHandler(svc)
	ctx := context.Background()

	_, err := h.GetBalance(ctx, &pb.GetBalanceRequest{AccountId: "not-a-uuid"})
	if err == nil {
		t.Fatal("expected error for invalid account_id")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}

	_, err = h.GetBalance(ctx, &pb.GetBalanceRequest{AccountId: uuid.New().String()})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
	st, _ = status.FromError(err)
	if st.Code() != codes.NotFound {
		t.Errorf("expected NotFound, got %v", st.Code())
	}
}

func TestPaymentHandler_Deposit_InvalidAccountAndSuccess(t *testing.T) {
	repo := testutil.NewFakeRepo()
	svc := service.NewPaymentService(repo, nil, nil)
	h := NewPaymentHandler(svc)
	ctx := context.Background()

	_, err := h.Deposit(ctx, &pb.DepositRequest{AccountId: "bad", Amount: 100})
	if err == nil {
		t.Fatal("expected error for invalid account_id")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}

	acc, _ := svc.CreateAccount(ctx, "u1")
	resp, err := h.Deposit(ctx, &pb.DepositRequest{
		AccountId: acc.ID.String(),
		Amount:    500,
	})
	if err != nil {
		t.Fatalf("Deposit: %v", err)
	}
	if resp.GetTransactionId() == "" || resp.GetNewBalance() != 500 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestPaymentHandler_Transfer_InvalidArgsAndInsufficientFunds(t *testing.T) {
	repo := testutil.NewFakeRepo()
	svc := service.NewPaymentService(repo, nil, nil)
	h := NewPaymentHandler(svc)
	ctx := context.Background()

	from, _ := svc.CreateAccount(ctx, "from")
	to, _ := svc.CreateAccount(ctx, "to")
	_, _, _ = svc.Deposit(ctx, from.ID, 100, "")

	_, err := h.Transfer(ctx, &pb.TransferRequest{
		FromAccountId: "bad",
		ToAccountId:   to.ID.String(),
		Amount:        10,
	})
	if err == nil {
		t.Fatal("expected error for invalid from_account_id")
	}
	st, _ := status.FromError(err)
	if st.Code() != codes.InvalidArgument {
		t.Errorf("expected InvalidArgument, got %v", st.Code())
	}

	// недостаточно средств
	_, err = h.Transfer(ctx, &pb.TransferRequest{
		FromAccountId: from.ID.String(),
		ToAccountId:   to.ID.String(),
		Amount:        200,
	})
	if err == nil {
		t.Fatal("expected error for insufficient funds")
	}
	st, _ = status.FromError(err)
	if st.Code() != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition, got %v", st.Code())
	}
}

func TestPaymentHandler_Transfer_Success(t *testing.T) {
	repo := testutil.NewFakeRepo()
	svc := service.NewPaymentService(repo, nil, nil)
	h := NewPaymentHandler(svc)
	ctx := context.Background()

	from, _ := svc.CreateAccount(ctx, "from")
	to, _ := svc.CreateAccount(ctx, "to")
	_, _, _ = svc.Deposit(ctx, from.ID, 100, "")

	resp, err := h.Transfer(ctx, &pb.TransferRequest{
		FromAccountId: from.ID.String(),
		ToAccountId:   to.ID.String(),
		Amount:        40,
	})
	if err != nil {
		t.Fatalf("Transfer: %v", err)
	}
	if !resp.GetSuccess() || resp.GetFromNewBalance() != 60 || resp.GetToNewBalance() != 40 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestToStatusError_mapsUnknownErrorToInternal(t *testing.T) {
	// Обычная ошибка без кода должна мапиться в Internal (важно для SDET: клиент не должен получать детали).
	err := toStatusError(errors.New("something broke"))
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatal("expected gRPC status")
	}
	if st.Code() != codes.Internal {
		t.Errorf("unknown error should map to Internal, got %v", st.Code())
	}
}

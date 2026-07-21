package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/moneymate-2026/moneymate-backend/shared/proto/merchant"
	"github.com/moneymate-2026/moneymate-backend/services/merchant/internal/usecases"
)
// MerchantHandler implements the gRPC server interface for merchant operations.
type MerchantHandler struct {
	pb.UnimplementedMerchantServiceServer
	usecase *usecases.StoreUseCase
}

// NewMerchantHandler injects the usecase dependency into the transport layer.
func NewMerchantHandler(uc *usecases.StoreUseCase) *MerchantHandler {
	return &MerchantHandler{usecase: uc}
}

// RegisterStore maps the gRPC request to the domain input and executes the workflow.
func (h *MerchantHandler) RegisterStore(ctx context.Context, req *pb.RegisterStoreRequest) (*pb.RegisterStoreResponse, error) {
	input := usecases.RegisterStoreInput{
		OwnerID:      req.GetOwnerId(), // Typically extracted from context metadata in production
		OwnerName:    req.GetOwnerName(),
		ContactEmail: req.GetContactEmail(),
		MobileNumber: req.GetMobileNumber(),
		LegalName:    req.GetLegalName(),
		Type:         req.GetType(),
	}

	// Handle optional pointer fields
	if req.GetDbaName() != "" {
		dba := req.GetDbaName()
		input.DBAName = &dba
	}
	if req.GetTaxId() != "" {
		taxID := req.GetTaxId()
		input.TaxID = &taxID
	}

	store, err := h.usecase.ProcessRegistration(ctx, input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process registration: %v", err)
	}

	return &pb.RegisterStoreResponse{
		StoreId:   store.ID.String(),
		DisplayId: store.DisplayID,
		Status:    store.Status,
		Plan:      store.Plan,
	}, nil
}
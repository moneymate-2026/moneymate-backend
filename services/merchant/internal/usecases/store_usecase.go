package usecases

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/moneymate-2026/moneymate-backend/services/merchant/internal/domain"
)

// StoreUseCase orchestrates merchant workflows.
type StoreUseCase struct {
	repo domain.MerchantRepository
}

// NewStoreUseCase constructs a usecase with repository dependencies.
func NewStoreUseCase(repo domain.MerchantRepository) *StoreUseCase {
	return &StoreUseCase{repo: repo}
}

type RegisterStoreInput struct {
	OwnerID           string
	OwnerName         string
	ContactEmail      string
	MobileNumber      string
	LegalName         string
	DBAName           *string
	Type              string
	TaxID             *string
	RegisteredAddress string
	AadhaarNumber     string
	AadhaarDocURL     string
	ShopLicenseURL    string
}

// ProcessRegistration applies validation and executes state persistence.
func (uc *StoreUseCase) ProcessRegistration(ctx context.Context, in RegisterStoreInput) (*domain.Store, error) {
	ownerUUID, err := uuid.Parse(in.OwnerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner UUID format: %w", err)
	}

	displayID, err := generateDisplayID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate secure display ID: %w", err)
	}

	store := &domain.Store{
		OwnerID:           ownerUUID,
		OwnerName:         strings.TrimSpace(in.OwnerName),
		ContactEmail:      strings.ToLower(strings.TrimSpace(in.ContactEmail)),
		MobileNumber:      strings.TrimSpace(in.MobileNumber),
		LegalName:         strings.TrimSpace(in.LegalName),
		DBAName:           in.DBAName,
		Type:              in.Type,
		TaxID:             in.TaxID,
		RegisteredAddress: strings.TrimSpace(in.RegisteredAddress),
		DisplayID:         displayID,
	}

	createdStore, err := uc.repo.RegisterStore(ctx, store)
	if err != nil {
		return nil, fmt.Errorf("failed to register store: %w", err)
	}

	kyc := &domain.KYCDocument{
		StoreID:        createdStore.ID,
		AadhaarNumber:  in.AadhaarNumber,
		AadhaarDocURL:  in.AadhaarDocURL,
		ShopLicenseURL: in.ShopLicenseURL,
	}

	if err := uc.repo.SubmitKYC(ctx, kyc); err != nil {
		return nil, fmt.Errorf("failed to submit KYC documents: %w", err)
	}

	return createdStore, nil
}

// GetStore retrieves a store by owner ID.
func (uc *StoreUseCase) GetStore(ctx context.Context, ownerID string) (*domain.Store, error) {
	ownerUUID, err := uuid.Parse(ownerID)
	if err != nil {
		return nil, fmt.Errorf("invalid owner UUID format: %w", err)
	}

	return uc.repo.GetStoreByOwnerID(ctx, ownerUUID)
}

// generateDisplayID yields a collision-resistant MM-XXXX-XX identifier.
func generateDisplayID() (string, error) {
	b := make([]byte, 3)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	hexStr := strings.ToUpper(hex.EncodeToString(b))
	return fmt.Sprintf("MM-%s-%s", hexStr[:4], hexStr[4:]), nil
}

// GetPendingStores retrieves all merchants in the pending_kyc status.
func (uc *StoreUseCase) GetPendingStores(ctx context.Context) ([]*domain.Store, error) {
	return uc.repo.GetPendingStores(ctx)
}
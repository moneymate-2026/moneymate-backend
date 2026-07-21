package repo

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/moneymate-2026/moneymate-backend/services/merchant/internal/domain"
	"github.com/moneymate-2026/moneymate-backend/services/merchant/sqlc/generated"
)

// StoreRepo implements domain.MerchantRepository using pgxpool for mechanical sympathy.
type StoreRepo struct {
	db *pgxpool.Pool
	q  *generated.Queries
}

// NewStoreRepo initializes the repository instance.
func NewStoreRepo(db *pgxpool.Pool) *StoreRepo {
	return &StoreRepo{
		db: db,
		q:  generated.New(db),
	}
}

// RegisterStore commits step 1 and 2 of the merchant onboarding flow.
func (r *StoreRepo) RegisterStore(ctx context.Context, store *domain.Store) (*domain.Store, error) {
	arg := generated.CreateStoreParams{
		OwnerID:      store.OwnerID,
		OwnerName:    store.OwnerName,
		ContactEmail: store.ContactEmail,
		MobileNumber: store.MobileNumber,
		LegalName:    store.LegalName,
		DbaName:      store.DBAName,
		Type:         generated.BusinessType(store.Type),
		TaxID:        store.TaxID,
		DisplayID:    store.DisplayID,
	}

	row, err := r.q.CreateStore(ctx, arg)
	if err != nil {
		return nil, fmt.Errorf("StoreRepo.RegisterStore insertion failed: %w", err)
	}

	store.ID = row.ID
	store.Status = string(row.Status)
	store.Plan = string(row.Plan)
	store.CreatedAt = row.CreatedAt

	return store, nil
}

// SubmitKYC commits step 3 compliance documents.
func (r *StoreRepo) SubmitKYC(ctx context.Context, kyc *domain.KYCDocument) error {
	arg := generated.SubmitKYCParams{
		StoreID:        kyc.StoreID,
		AadhaarNumber:  kyc.AadhaarNumber,
		AadhaarDocUrl:  kyc.AadhaarDocURL,
		ShopLicenseUrl: kyc.ShopLicenseURL,
	}

	if err := r.q.SubmitKYC(ctx, arg); err != nil {
		return fmt.Errorf("StoreRepo.SubmitKYC failed: %w", err)
	}
	return nil
}

// GetStoreByOwnerID retrieves the store state for gateway routing.
func (r *StoreRepo) GetStoreByOwnerID(ctx context.Context, ownerID uuid.UUID) (*domain.Store, error) {
	row, err := r.q.GetStoreByOwnerID(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("StoreRepo.GetStoreByOwnerID query failed: %w", err)
	}

	return &domain.Store{
		ID:        row.ID,
		DisplayID: row.DisplayID,
		LegalName: row.LegalName,
		Status:    string(row.Status),
		Plan:      string(row.Plan),
	}, nil
}

// UpdateStoreStatus advances the state machine (e.g., pending_kyc -> active).
func (r *StoreRepo) UpdateStoreStatus(ctx context.Context, storeID uuid.UUID, status string) error {
	arg := generated.UpdateStoreStatusParams{
		ID:     storeID,
		Status: generated.MerchantStatus(status),
	}

	if err := r.q.UpdateStoreStatus(ctx, arg); err != nil {
		return fmt.Errorf("StoreRepo.UpdateStoreStatus failed: %w", err)
	}
	return nil
}
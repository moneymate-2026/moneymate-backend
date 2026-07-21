package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Store represents the core merchant entity.
type Store struct {
	ID           uuid.UUID
	OwnerID      uuid.UUID
	CreatedAt    time.Time
	UpdatedAt    time.Time
	OwnerName    string
	ContactEmail string
	MobileNumber string
	LegalName    string
	DBAName      *string
	Type         string
	TaxID        *string
	DisplayID    string
	Status       string
	Plan         string
}

// KYCDocument represents compliance data.
type KYCDocument struct {
	ID             uuid.UUID
	StoreID        uuid.UUID
	VerifiedAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
	AadhaarNumber  string
	AadhaarDocURL  string
	ShopLicenseURL string
	IsVerified     bool
}

// MerchantRepository defines the strict data access contract.
type MerchantRepository interface {
	RegisterStore(ctx context.Context, store *Store) (*Store, error)
	SubmitKYC(ctx context.Context, kyc *KYCDocument) error
	GetStoreByOwnerID(ctx context.Context, ownerID uuid.UUID) (*Store, error)
	UpdateStoreStatus(ctx context.Context, storeID uuid.UUID, status string) error
}
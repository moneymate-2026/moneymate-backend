package http

import (
	"github.com/gofiber/fiber/v3"
	"github.com/moneymate-2026/moneymate-backend/services/merchant/internal/usecases"
)

type MerchantHandler struct {
	usecase *usecases.StoreUseCase
}

func NewMerchantHandler(uc *usecases.StoreUseCase) *MerchantHandler {
	return &MerchantHandler{usecase: uc}
}

func (h *MerchantHandler) RegisterStore(c fiber.Ctx) error {
	var req RegisterStoreRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "Invalid request body"})
	}

	input := usecases.RegisterStoreInput{
		OwnerID:           req.OwnerID,
		OwnerName:         req.OwnerName,
		ContactEmail:      req.ContactEmail,
		MobileNumber:      req.MobileNumber,
		LegalName:         req.LegalName,
		Type:              req.BusinessType,
		RegisteredAddress: req.RegisteredAddress,
		AadhaarNumber:     req.AadhaarNumber,
		AadhaarDocURL:     req.AadhaarDocURL,
		ShopLicenseURL:    req.ShopLicenseURL,
	}

	if req.DBAName != "" {
		input.DBAName = &req.DBAName
	}
	if req.TaxID != "" {
		input.TaxID = &req.TaxID
	}

	store, err := h.usecase.ProcessRegistration(c.Context(), input)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(RegisterStoreResponse{
		StoreID:   store.ID.String(),
		DisplayID: store.DisplayID,
		Status:    store.Status,
		Plan:      store.Plan,
	})
}

func (h *MerchantHandler) GetStore(c fiber.Ctx) error {
	ownerID := c.Params("owner_id")
	if ownerID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "owner_id is required"})
	}

	store, err := h.usecase.GetStore(c.Context(), ownerID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(GetStoreResponse{
		StoreID:   store.ID.String(),
		DisplayID: store.DisplayID,
		Status:    store.Status,
		Plan:      store.Plan,
		LegalName: store.LegalName,
	})
}

func (h *MerchantHandler) GetPendingStores(c fiber.Ctx) error {
	stores, err := h.usecase.GetPendingStores(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	
	// Convert domain.Store to DTOs or anonymous structs here if necessary
	var responseList []fiber.Map
	for _, s := range stores {
		responseList = append(responseList, fiber.Map{
			"store_id":       s.ID.String(),
			"owner_name":     s.OwnerName,
			"contact_email":  s.ContactEmail,
			"mobile_number":  s.MobileNumber,
			"legal_name":     s.LegalName,
			"business_type":  s.Type,
			"status":         s.Status,
			"created_at":     s.CreatedAt,
		})
	}

	return c.JSON(fiber.Map{
		"stores": responseList,
	})
}

package http

type RegisterStoreRequest struct {
	OwnerID           string `json:"owner_id"`
	OwnerName         string `json:"owner_name"`
	ContactEmail      string `json:"contact_email"`
	MobileNumber      string `json:"mobile_number"`
	LegalName         string `json:"legal_name"`
	DBAName           string `json:"dba_name,omitempty"`
	BusinessType      string `json:"business_type"`
	TaxID             string `json:"tax_id,omitempty"`
	RegisteredAddress string `json:"registered_address"`
	AadhaarNumber     string `json:"aadhaar_number"`
	AadhaarDocURL     string `json:"aadhaar_doc_url"`
	ShopLicenseURL    string `json:"shop_license_url"`
}

type RegisterStoreResponse struct {
	StoreID   string `json:"store_id"`
	DisplayID string `json:"display_id"`
	Status    string `json:"status"`
	Plan      string `json:"plan"`
}

type GetStoreResponse struct {
	StoreID   string `json:"store_id"`
	DisplayID string `json:"display_id"`
	Status    string `json:"status"`
	Plan      string `json:"plan"`
	LegalName string `json:"legal_name"`
}

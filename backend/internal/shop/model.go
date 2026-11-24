package shop

// Shop is the seller's shop profile as returned by the API.
// Optional fields are null when unset.
type Shop struct {
	ID            int64   `json:"id"`
	Name          string  `json:"name"`
	LogoURL       *string `json:"logo_url"`
	Description   string  `json:"description"`
	ContactPhone  *string `json:"contact_phone"`
	PickupAddress *string `json:"pickup_address"`
}

// UpdateRequest is the body of PUT /shop. It replaces all editable fields.
// An empty optional field clears the stored value.
type UpdateRequest struct {
	Name          string `json:"name"`
	LogoURL       string `json:"logo_url"`
	Description   string `json:"description"`
	ContactPhone  string `json:"contact_phone"`
	PickupAddress string `json:"pickup_address"`
}

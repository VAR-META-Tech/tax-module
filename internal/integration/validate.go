package integration

import (
	"fmt"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ValidateViettelRequest validates a ViettelInvoiceRequest against Viettel API constraints
// defined via `validate` struct tags before sending to the Viettel API.
func ValidateViettelRequest(req *ViettelInvoiceRequest) error {
	if err := validate.Struct(req); err != nil {
		return fmt.Errorf("viettel request validation failed: %w", err)
	}
	return nil
}

package apperror_test

import (
	"errors"
	"fmt"
	"testing"
	"{{ .Computed.module_name_final }}/internal/common/apperror"
)

func TestPublicErrors(t *testing.T) {
	expected := apperror.New("EXAMPLE_NOT_FOUND", "example not found")
	wrapped := fmt.Errorf("lookup: %w", expected)
	if !errors.Is(wrapped, expected) || apperror.Code(wrapped) != "EXAMPLE_NOT_FOUND" || apperror.Public(wrapped).Error() != "example not found" {
		t.Fatal("wrapped error lost identity")
	}
	for _, err := range []error{nil, errors.New("database secret"), (*apperror.Error)(nil)} {
		if apperror.Public(err) != apperror.Internal {
			t.Fatal("unexpected error exposed")
		}
	}
}

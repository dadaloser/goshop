package user

import (
	"testing"

	srvv1 "goshop/app/user/srv/internal/service/v1"
)

func TestDTOToResponseIncludesAccountStatus(t *testing.T) {
	response := DTOToResponse(srvv1.UserPublicDTO{Status: "disabled"})

	if response.Status != "disabled" {
		t.Fatalf("status = %q, want disabled", response.Status)
	}
}

package user

import (
	"testing"

	datav1 "goshop/app/user/srv/internal/data/v1"
	srvv1 "goshop/app/user/srv/internal/service/v1"
)

func TestDTOToResponseIncludesAccountStatus(t *testing.T) {
	response := DTOToResponse(srvv1.UserDTO{UserDO: datav1.UserDO{Status: "disabled"}})

	if response.Status != "disabled" {
		t.Fatalf("status = %q, want disabled", response.Status)
	}
}

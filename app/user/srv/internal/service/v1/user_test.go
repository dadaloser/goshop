package v1

import (
	"context"
	"goshop/app/user/srv/internal/data/v1/mock"
	metav1 "goshop/pkg/common/meta/v1"

	"testing"
)

func TestUserList(t *testing.T) {
	userSrv := NewUserService(mock.NewUsers())
	userSrv.List(context.Background(), nil, metav1.ListMeta{})
}

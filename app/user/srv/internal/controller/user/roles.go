package user

import (
	"context"

	upbv1 "goshop/api/user/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/errors"

	"google.golang.org/protobuf/types/known/emptypb"
)

func (u *userServer) ListStaffRoles(ctx context.Context, _ *emptypb.Empty) (*upbv1.StaffRoleListResponse, error) {
	roles, err := u.srv.ListStaffRoles(ctx)
	if err != nil {
		return nil, err
	}

	response := &upbv1.StaffRoleListResponse{}
	for _, role := range roles {
		response.Roles = append(response.Roles, &upbv1.StaffRole{
			Name:        role.Name,
			Description: role.Description,
			Permissions: append([]string(nil), role.Permissions...),
		})
	}
	return response, nil
}

func (u *userServer) GetUserStaffRoles(ctx context.Context, request *upbv1.IdRequest) (*upbv1.UserRoleBindingResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "user id request is required")
	}

	binding, err := u.srv.GetUserRoleBinding(ctx, uint64(request.Id))
	if err != nil {
		return nil, err
	}
	return &upbv1.UserRoleBindingResponse{
		UserId:      binding.UserID,
		Roles:       append([]string(nil), binding.StaffRoles...),
		Permissions: append([]string(nil), binding.Permissions...),
	}, nil
}

func (u *userServer) ReplaceUserStaffRoles(ctx context.Context, request *upbv1.ReplaceUserStaffRolesRequest) (*upbv1.UserRoleBindingResponse, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "replace user staff roles request is required")
	}

	binding, err := u.srv.ReplaceUserRoleBinding(ctx, uint64(request.UserId), request.Roles)
	if err != nil {
		return nil, err
	}
	return &upbv1.UserRoleBindingResponse{
		UserId:      binding.UserID,
		Roles:       append([]string(nil), binding.StaffRoles...),
		Permissions: append([]string(nil), binding.Permissions...),
	}, nil
}

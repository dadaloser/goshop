package user

import (
	"context"

	upbv1 "goshop/api/user/v1"
	srvv1 "goshop/app/user/srv/internal/service/v1"
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
			Builtin:     role.Builtin,
			Domains:     append([]string(nil), role.Domains...),
		})
	}
	return response, nil
}

func (u *userServer) CreateStaffRole(ctx context.Context, request *upbv1.CreateStaffRoleRequest) (*upbv1.StaffRole, error) {
	if request == nil || request.Role == nil {
		return nil, errors.WithCode(code2.ErrValidation, "create staff role request is required")
	}

	role, err := u.srv.CreateStaffRole(ctx, srvv1.StaffRoleDTO{
		Name:        request.Role.Name,
		Description: request.Role.Description,
		Permissions: append([]string(nil), request.Role.Permissions...),
		Domains:     append([]string(nil), request.Role.Domains...),
	})
	if err != nil {
		return nil, err
	}
	return &upbv1.StaffRole{
		Name:        role.Name,
		Description: role.Description,
		Permissions: append([]string(nil), role.Permissions...),
		Builtin:     role.Builtin,
		Domains:     append([]string(nil), role.Domains...),
	}, nil
}

func (u *userServer) UpdateStaffRole(ctx context.Context, request *upbv1.UpdateStaffRoleRequest) (*upbv1.StaffRole, error) {
	if request == nil || request.Role == nil {
		return nil, errors.WithCode(code2.ErrValidation, "update staff role request is required")
	}

	role, err := u.srv.UpdateStaffRole(ctx, srvv1.StaffRoleDTO{
		Name:        request.Role.Name,
		Description: request.Role.Description,
		Permissions: append([]string(nil), request.Role.Permissions...),
		Domains:     append([]string(nil), request.Role.Domains...),
	})
	if err != nil {
		return nil, err
	}
	return &upbv1.StaffRole{
		Name:        role.Name,
		Description: role.Description,
		Permissions: append([]string(nil), role.Permissions...),
		Builtin:     role.Builtin,
		Domains:     append([]string(nil), role.Domains...),
	}, nil
}

func (u *userServer) DeleteStaffRole(ctx context.Context, request *upbv1.DeleteStaffRoleRequest) (*emptypb.Empty, error) {
	if request == nil {
		return nil, errors.WithCode(code2.ErrValidation, "delete staff role request is required")
	}
	if err := u.srv.DeleteStaffRole(ctx, request.GetName()); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
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

	binding, err := u.srv.ReplaceUserRoleBinding(ctx, uint64(request.UserId), request.Roles, auditActorFromProto(request.Actor))
	if err != nil {
		return nil, err
	}
	return &upbv1.UserRoleBindingResponse{
		UserId:      binding.UserID,
		Roles:       append([]string(nil), binding.StaffRoles...),
		Permissions: append([]string(nil), binding.Permissions...),
	}, nil
}

package user

import (
	"context"

	upbv1 "goshop/api/user/v1"
	srvv1 "goshop/app/user/srv/internal/service/v1"
)

func (s *userServer) ReplaceUserResourceScopes(ctx context.Context, req *upbv1.ReplaceUserResourceScopesRequest) (*upbv1.UserResourceScopeListResponse, error) {
	scopes := make([]srvv1.ResourceScopeDTO, 0, len(req.GetScopes()))
	for _, scope := range req.GetScopes() {
		scopes = append(scopes, srvv1.ResourceScopeDTO{Domain: scope.GetDomain(), StoreID: scope.GetStoreId(), TeamID: scope.GetTeamId()})
	}
	replaced, err := s.srv.ReplaceResourceScopes(ctx, uint64(req.GetUserId()), scopes)
	if err != nil {
		return nil, err
	}
	resp := &upbv1.UserResourceScopeListResponse{UserId: req.GetUserId(), Scopes: make([]*upbv1.UserResourceScope, 0, len(replaced))}
	for _, scope := range replaced {
		resp.Scopes = append(resp.Scopes, &upbv1.UserResourceScope{Domain: scope.Domain, StoreId: scope.StoreID, TeamId: scope.TeamID})
	}
	return resp, nil
}

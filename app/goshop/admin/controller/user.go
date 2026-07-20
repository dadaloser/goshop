package controller

import (
	upbv1 "goshop/api/user/v1"
	"goshop/app/pkg/authsession/tokenversion"
)

type userServer struct {
	users         upbv1.UserClient
	tokenVersions tokenversion.Store
}

func NewUserController(users upbv1.UserClient, tokenVersions tokenversion.Store) *userServer {
	return &userServer{users: users, tokenVersions: tokenVersions}
}

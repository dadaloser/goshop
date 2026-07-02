package v1

import (
	"context"
	"goshop/app/pkg/code"
	"goshop/pkg/errors"
	"goshop/pkg/log"
	"time"

	"goshop/app/goshop/api/internal/data"
	"goshop/app/goshop/api/internal/smscode"
	"goshop/app/pkg/options"
	"goshop/gmicro/server/restserver/middlewares"

	"github.com/golang-jwt/jwt/v5"
)

type UserDTO struct {
	data.User

	Token     string `json:"token"`
	ExpiresAt int64  `json:"expires_at"`
}

type UserSrv interface {
	MobileLogin(ctx context.Context, mobile, password string) (*UserDTO, error)
	Register(ctx context.Context, mobile, password, code string) (*UserDTO, error)
	Update(ctx context.Context, userDTO *UserDTO) error
	Get(ctx context.Context, userID uint64) (*UserDTO, error)
	GetByMobile(ctx context.Context, mobile string) (*UserDTO, error)
	CheckPassWord(ctx context.Context, password, EncryptedPassword string) (bool, error)
}

type userService struct {
	//ud data.UserData
	data data.DataFactory

	jwtOpts *options.JwtOptions

	codeStore smscode.Store
}

func NewUserService(data data.DataFactory, jwtOpts *options.JwtOptions, codeStore smscode.Store) UserSrv {
	return &userService{data: data, jwtOpts: jwtOpts, codeStore: codeStore}
}

func (us *userService) MobileLogin(ctx context.Context, mobile, password string) (*UserDTO, error) {
	user, err := us.data.Users().GetByMobile(ctx, mobile)
	if err != nil {
		if errors.IsCode(err, code.ErrUserNotFound) {
			return nil, errors.WithCode(code.ErrUserPasswordIncorrect, "手机号或密码错误")
		}
		return nil, err
	}

	//检查密码是否正确
	err = us.data.Users().CheckPassWord(ctx, password, user.PassWord)
	if err != nil {
		if errors.IsCode(err, code.ErrUserPasswordIncorrect) {
			return nil, errors.WithCode(code.ErrUserPasswordIncorrect, "手机号或密码错误")
		}
		return nil, err
	}

	token, expiresAt, err := us.createToken(user)
	if err != nil {
		return nil, err
	}

	return &UserDTO{
		User:      user,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (us *userService) Register(ctx context.Context, mobile, password, codes string) (*UserDTO, error) {
	key := smscode.RegisterKey(mobile)
	value, err := us.codeStore.Get(ctx, key)
	if err != nil {
		return nil, errors.WithCode(code.ErrCodeNotExist, "验证码不存在")
	}

	if value != codes {
		return nil, errors.WithCode(code.ErrCodeInCorrect, "验证码错误")
	}

	var user = &data.User{
		Mobile:   mobile,
		PassWord: password,
	}
	err = us.data.Users().Create(ctx, user)
	if err != nil {
		log.Errorf("user register failed: %v", err)
		return nil, err
	}

	if ok := us.codeStore.Delete(ctx, key); !ok {
		log.Warn("delete sms code failed")
	}

	token, expiresAt, err := us.createToken(*user)
	if err != nil {
		return nil, err
	}

	return &UserDTO{
		User:      *user,
		Token:     token,
		ExpiresAt: expiresAt,
	}, nil
}

func (us *userService) createToken(user data.User) (string, int64, error) {
	now := time.Now()
	j := middlewares.NewJWT(us.jwtOpts.Key)
	claims := middlewares.CustomClaims{
		ID:          uint(user.ID),
		NickName:    user.NickName,
		AuthorityId: uint(user.Role),
		RegisteredClaims: jwt.RegisteredClaims{
			NotBefore: jwt.NewNumericDate(now), //签名的生效时间
			ExpiresAt: jwt.NewNumericDate(now.Add(us.jwtOpts.Timeout)),
			Issuer:    us.jwtOpts.Realm,
		},
	}
	token, err := j.CreateToken(claims)
	if err != nil {
		return "", 0, err
	}
	return token, now.Local().Add(us.jwtOpts.Timeout).Unix(), nil
}

func (u *userService) Update(ctx context.Context, userDTO *UserDTO) error {
	//TODO implement me
	panic("implement me")
}

func (us *userService) Get(ctx context.Context, userID uint64) (*UserDTO, error) {
	userDO, err := us.data.Users().Get(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &UserDTO{User: userDO}, nil
}

func (u *userService) GetByMobile(ctx context.Context, mobile string) (*UserDTO, error) {
	//TODO implement me
	panic("implement me")
}

func (u *userService) CheckPassWord(ctx context.Context, password, EncryptedPassword string) (bool, error) {
	//TODO implement me
	panic("implement me")
}

var _ UserSrv = &userService{}

package v1

import (
	"context"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"goshop/app/pkg/authz"
	"goshop/app/pkg/code"
	dv1 "goshop/app/user/srv/internal/data/v1"
	code2 "goshop/gmicro/code"
	"goshop/pkg/common/auth"
	metav1 "goshop/pkg/common/meta/v1"
	"goshop/pkg/errors"
)

type UserDTO struct {
	//正常要重新写,不是直接引用
	dv1.UserDO
}

type UserPublicDTO struct {
	ID         int32
	Username   *string
	Mobile     string
	Email      *string
	NickName   string
	Birthday   *time.Time
	Gender     string
	LegacyRole int32
	Status     string
}

type UserAuthDTO struct {
	UserPublicDTO
	PasswordHash string
	StaffRoles   []string
	Permissions  []string
}

type StaffRoleDTO struct {
	Name        string
	Description string
	Permissions []string
}

type UserRoleBindingDTO struct {
	UserID      int32
	StaffRoles  []string
	Permissions []string
}

type UserSrv interface {
	List(ctx context.Context, orderBy []string, opts metav1.ListMeta) (*UserPublicDTOList, error)
	Create(ctx context.Context, user *UserDTO) error
	Update(ctx context.Context, user *UserDTO) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, ID uint64) (*UserPublicDTO, error)
	GetByMobile(ctx context.Context, mobile string) (*UserPublicDTO, error)
	GetByUsername(ctx context.Context, username string) (*UserPublicDTO, error)
	UpdateStatus(ctx context.Context, userID uint64, status string) (*UserPublicDTO, error)
	GetAuthByID(ctx context.Context, ID uint64) (*UserAuthDTO, error)
	GetAuthByUsername(ctx context.Context, username string) (*UserAuthDTO, error)
	ListStaffRoles(ctx context.Context) ([]StaffRoleDTO, error)
	GetUserRoleBinding(ctx context.Context, userID uint64) (*UserRoleBindingDTO, error)
	ReplaceUserRoleBinding(ctx context.Context, userID uint64, roleNames []string) (*UserRoleBindingDTO, error)
}

type userService struct {
	userStore dv1.UserStore
}

var (
	usernamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{2,31}$`)
	mobilePattern   = regexp.MustCompile(`^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$`)
)

const (
	minPasswordLength = 8
	maxPasswordBytes  = 72
)

func (u *userService) Create(ctx context.Context, user *UserDTO) error {
	if user == nil {
		return errors.WithCode(code2.ErrValidation, "用户信息不能为空")
	}
	if err := normalizeUserIdentifiers(user); err != nil {
		return err
	}
	if user.Role == 0 {
		user.Role = int(authz.LegacyUserRoleCustomer)
	}
	if err := validateLegacyRole(int32(user.Role)); err != nil {
		return err
	}
	if !mobilePattern.MatchString(user.Mobile) {
		return errors.WithCode(code2.ErrValidation, "手机号格式错误")
	}
	if !isStrongPassword(user.Password) {
		return errors.WithCode(code2.ErrValidation, "密码必须为8-72字节，并包含大小写字母、数字和特殊字符")
	}
	if strings.TrimSpace(user.Status) == "" {
		user.Status = string(authz.AccountStatusActive)
	}

	//先判断用户是否存在
	if _, err := u.userStore.GetByMobile(ctx, user.Mobile); err == nil {
		return errors.WithCode(code.ErrUserAlreadyExists, "用户已经存在")
	} else if !errors.IsCode(err, code.ErrUserNotFound) {
		return err
	}

	if user.Email != nil {
		if _, err := u.userStore.GetByUsername(ctx, *user.Email); err == nil {
			return errors.WithCode(code.ErrUserAlreadyExists, "邮箱已经存在")
		} else if !errors.IsCode(err, code.ErrUserNotFound) {
			return err
		}
	}

	if user.Username != nil {
		if _, err := u.userStore.GetByUsername(ctx, *user.Username); err == nil {
			return errors.WithCode(code.ErrUserAlreadyExists, "用户名已经存在")
		} else if !errors.IsCode(err, code.ErrUserNotFound) {
			return err
		}
	}

	encryptedPassword, err := auth.Encrypt(user.Password)
	if err != nil {
		return errors.WithCode(code2.ErrEncrypt, "加密密码失败")
	}
	user.Password = encryptedPassword
	return u.userStore.Create(ctx, &user.UserDO)
}

func (u *userService) Update(ctx context.Context, user *UserDTO) error {
	if user == nil {
		return errors.WithCode(code2.ErrValidation, "用户信息不能为空")
	}
	if err := normalizeUserIdentifiers(user); err != nil {
		return err
	}

	//先查询用户是否存在
	existing, err := u.userStore.GetByID(ctx, uint64(user.ID))
	if err != nil {
		return err
	}
	if user.Role == 0 && existing != nil {
		user.Role = existing.Role
	}
	if err := validateLegacyRole(int32(user.Role)); err != nil {
		return err
	}

	return u.userStore.Update(ctx, &user.UserDO)
}

func (u *userService) Delete(ctx context.Context, id uint64) error {
	if id == 0 {
		return errors.WithCode(code2.ErrValidation, "用户不存在")
	}
	return u.userStore.Delete(ctx, id)
}

func (u *userService) GetByID(ctx context.Context, ID uint64) (*UserPublicDTO, error) {
	userDO, err := u.userStore.GetByID(ctx, ID)
	if err != nil {
		return nil, err
	}
	return newUserPublicDTO(userDO)
}

func (u *userService) GetByMobile(ctx context.Context, mobile string) (*UserPublicDTO, error) {
	userDO, err := u.userStore.GetByMobile(ctx, mobile)
	if err != nil {
		return nil, err
	}
	return newUserPublicDTO(userDO)
}

func (u *userService) GetByUsername(ctx context.Context, username string) (*UserPublicDTO, error) {
	username = normalizeLoginIdentifier(username)
	if username == "" {
		return nil, errors.WithCode(code2.ErrValidation, "登录标识不能为空")
	}

	userDO, err := u.userStore.GetByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return newUserPublicDTO(userDO)
}

func (u *userService) UpdateStatus(ctx context.Context, userID uint64, status string) (*UserPublicDTO, error) {
	if userID == 0 {
		return nil, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	normalizedStatus := authz.NormalizeAccountStatus(status)
	switch normalizedStatus {
	case authz.AccountStatusActive, authz.AccountStatusDisabled, authz.AccountStatusLocked:
	default:
		return nil, errors.WithCode(code2.ErrValidation, "account status must be one of: active, disabled, locked")
	}

	if err := u.userStore.UpdateStatus(ctx, userID, string(normalizedStatus)); err != nil {
		return nil, err
	}
	return u.GetByID(ctx, userID)
}

func (u *userService) GetAuthByID(ctx context.Context, ID uint64) (*UserAuthDTO, error) {
	userDO, err := u.userStore.GetAuthByID(ctx, ID)
	if err != nil {
		return nil, err
	}
	return newUserAuthDTO(userDO)
}

func (u *userService) GetAuthByUsername(ctx context.Context, username string) (*UserAuthDTO, error) {
	username = normalizeLoginIdentifier(username)
	if username == "" {
		return nil, errors.WithCode(code2.ErrValidation, "登录标识不能为空")
	}

	userDO, err := u.userStore.GetAuthByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return newUserAuthDTO(userDO)
}

func (u *userService) ListStaffRoles(ctx context.Context) ([]StaffRoleDTO, error) {
	roles, err := u.userStore.ListRoles(ctx)
	if err != nil {
		return nil, err
	}

	permissionsByRole := make(map[string][]string)
	for _, definition := range authz.BuiltinRoleDefinitions() {
		values := make([]string, 0, len(definition.Permissions))
		for _, permission := range definition.Permissions {
			values = append(values, string(permission))
		}
		permissionsByRole[string(definition.Name)] = values
	}

	result := make([]StaffRoleDTO, 0, len(roles))
	for _, role := range roles {
		result = append(result, StaffRoleDTO{
			Name:        role.Name,
			Description: role.Description,
			Permissions: append([]string(nil), permissionsByRole[role.Name]...),
		})
	}
	return result, nil
}

func (u *userService) GetUserRoleBinding(ctx context.Context, userID uint64) (*UserRoleBindingDTO, error) {
	authUser, err := u.userStore.GetAuthByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return newUserRoleBindingDTO(authUser)
}

func (u *userService) ReplaceUserRoleBinding(ctx context.Context, userID uint64, roleNames []string) (*UserRoleBindingDTO, error) {
	for _, roleName := range roleNames {
		normalized := strings.ToLower(strings.TrimSpace(roleName))
		if normalized == "" {
			continue
		}
		if !authz.IsValidStaffRole(normalized) {
			return nil, errors.WithCode(code2.ErrValidation, "staff roles contain unknown values")
		}
	}
	authUser, err := u.userStore.ReplaceUserRoles(ctx, userID, roleNames)
	if err != nil {
		return nil, err
	}
	return newUserRoleBindingDTO(authUser)
}

func NewUserService(us dv1.UserStore) UserSrv {
	return &userService{
		userStore: us,
	}
}

func (u *UserDTO) UsernameValue() string {
	if u == nil || u.Username == nil {
		return ""
	}
	return *u.Username
}

func (u *UserDTO) EmailValue() string {
	if u == nil || u.Email == nil {
		return ""
	}
	return *u.Email
}

func normalizeUserIdentifiers(user *UserDTO) error {
	user.Mobile = strings.TrimSpace(user.Mobile)
	user.Username = normalizeUsername(user.UsernameValue())
	user.Email = normalizeEmail(user.EmailValue())

	if user.Username != nil && !usernamePattern.MatchString(*user.Username) {
		return errors.WithCode(code2.ErrValidation, "用户名格式错误")
	}
	if user.Email != nil && !isEmailAddress(*user.Email) {
		return errors.WithCode(code2.ErrValidation, "邮箱格式错误")
	}
	return nil
}

func normalizeUsername(value string) *string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return nil
	}
	return &value
}

func normalizeEmail(value string) *string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return nil
	}
	return &value
}

func normalizeLoginIdentifier(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	return value
}

func isEmailAddress(value string) bool {
	address, err := mail.ParseAddress(value)
	return err == nil && address.Name == "" && address.Address == value
}

func isStrongPassword(password string) bool {
	if utf8.RuneCountInString(password) < minPasswordLength || len(password) > maxPasswordBytes {
		return false
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool
	for _, r := range password {
		switch {
		case unicode.IsSpace(r):
			return false
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			hasSpecial = true
		default:
			hasSpecial = true
		}
	}
	return hasUpper && hasLower && hasDigit && hasSpecial
}

var _ UserSrv = &userService{}

type UserDTOList struct {
	TotalCount int64      `json:"totalCount,omitempty"` //总数
	Items      []*UserDTO `json:"data"`                 //数据
}

type UserPublicDTOList struct {
	TotalCount int64            `json:"totalCount,omitempty"`
	Items      []*UserPublicDTO `json:"data"`
}

func (u *userService) List(ctx context.Context, orderBy []string, opts metav1.ListMeta) (*UserPublicDTOList, error) {
	//这里是业务逻辑1
	/*
		1. data层的接口必须先写好
		2. 我期望测试的时候每次测试底层的data层的数据按照我期望的返回
			1. 先手动去插入一些数据
			2. 去删除一些数据
		3. 如果data层的方法有bug， 坑爹， 我们的代码想要具备好的可测试性
	*/

	doList, err := u.userStore.List(ctx, orderBy, opts)
	if err != nil {
		return nil, err
	}

	//业务逻辑2
	//代码不方便写单元测试用例
	userDTOList := UserPublicDTOList{TotalCount: doList.TotalCount}
	for _, value := range doList.Items {
		projectDTO, err := newUserPublicDTO(value)
		if err != nil {
			return nil, err
		}
		userDTOList.Items = append(userDTOList.Items, projectDTO)
	}

	//业务逻辑3
	return &userDTOList, nil
}

func newUserPublicDTO(user *dv1.UserDO) (*UserPublicDTO, error) {
	if user == nil {
		return nil, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}
	if err := validateLegacyRole(int32(user.Role)); err != nil {
		return nil, err
	}
	var birthday *time.Time
	if user.Birthday != nil {
		value := *user.Birthday
		birthday = &value
	}
	return &UserPublicDTO{
		ID:         user.ID,
		Username:   user.Username,
		Mobile:     user.Mobile,
		Email:      user.Email,
		NickName:   user.NickName,
		Birthday:   birthday,
		Gender:     user.Gender,
		LegacyRole: int32(user.Role),
		Status:     user.Status,
	}, nil
}

func newUserAuthDTO(user *dv1.UserAuthDO) (*UserAuthDTO, error) {
	if user == nil {
		return nil, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}
	publicDTO, err := newUserPublicDTO(&user.UserDO)
	if err != nil {
		return nil, err
	}
	return &UserAuthDTO{
		UserPublicDTO: *publicDTO,
		PasswordHash:  user.Password,
		StaffRoles:    append([]string(nil), user.StaffRoles...),
		Permissions:   append([]string(nil), user.Permissions...),
	}, nil
}

func PublicDTOFromMutation(user *UserDTO) (*UserPublicDTO, error) {
	if user == nil {
		return nil, errors.WithCode(code2.ErrValidation, "用户信息不能为空")
	}
	return newUserPublicDTO(&user.UserDO)
}

func newUserRoleBindingDTO(user *dv1.UserAuthDO) (*UserRoleBindingDTO, error) {
	if user == nil {
		return nil, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}
	return &UserRoleBindingDTO{
		UserID:      user.ID,
		StaffRoles:  append([]string(nil), user.StaffRoles...),
		Permissions: append([]string(nil), user.Permissions...),
	}, nil
}

func validateLegacyRole(role int32) error {
	if !authz.IsValidLegacyUserRole(role) {
		return errors.WithCode(code2.ErrValidation, "legacy role must be one of: 1(customer), 2(admin)")
	}
	return nil
}

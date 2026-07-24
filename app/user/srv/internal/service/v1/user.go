package v1

import (
	"context"
	"github.com/google/uuid"
	"net/mail"
	"regexp"
	"sort"
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
	ID             int32
	Username       *string
	Mobile         string
	Email          *string
	NickName       string
	Birthday       *time.Time
	Gender         string
	Status         string
	MobileVerified bool
	EmailVerified  bool
	LastLoginAt    *time.Time
}

type UserAuthDTO struct {
	UserPublicDTO
	PasswordHash    string
	LegacyRole      int32
	StaffRoles      []string
	Permissions     []string
	ResourceDomains []string
	ResourceStores  []string
	ResourceTeams   []string
}

type StaffRoleDTO struct {
	Name        string
	Description string
	Permissions []string
	Builtin     bool
	Domains     []string
}

type UserRoleBindingDTO struct {
	UserID      int32
	StaffRoles  []string
	Permissions []string
}

type AuditActorDTO struct {
	UserID        int32
	PrincipalType string
}

type StaffUserDTO struct {
	User        UserPublicDTO
	StaffRoles  []string
	Permissions []string
}

type UserAuditLogDTO struct {
	ID                 uint64
	UserID             int32
	ActorUserID        int32
	ActorPrincipalType string
	Action             string
	FromStatus         string
	ToStatus           string
	Detail             string
	CreatedAt          time.Time
}

type UserAuditLogDTOList struct {
	TotalCount int64
	Items      []*UserAuditLogDTO
}

type UserAuditLogFilterDTO struct {
	Action             string
	ActorUserID        int32
	ActorPrincipalType string
	CreatedAfter       *time.Time
	CreatedBefore      *time.Time
}

type AdminAuditLogDTO struct {
	ID                 uint64
	TargetUserID       int32
	ActorUserID        int32
	ActorPrincipalType string
	Action             string
	Detail             string
	CorrelationID      string
	RequestID          string
	TargetType         string
	TargetID           string
	Domain             string
	StoreID            string
	TeamID             string
	CreatedAt          time.Time
}

type AdminAuditLogDTOList struct {
	TotalCount int64
	Items      []*AdminAuditLogDTO
}

type AdminAuditLogFilterDTO struct {
	TargetUserID       int32
	Action             string
	ActorUserID        int32
	ActorPrincipalType string
	CreatedAfter       *time.Time
	CreatedBefore      *time.Time
}

type UserSrv interface {
	List(ctx context.Context, orderBy []string, opts metav1.ListMeta) (*UserPublicDTOList, error)
	Create(ctx context.Context, user *UserDTO) error
	CreateStaff(ctx context.Context, user *UserDTO, roleNames []string, status string, actor AuditActorDTO) (*StaffUserDTO, error)
	Update(ctx context.Context, user *UserDTO) error
	Delete(ctx context.Context, id uint64) error
	GetByID(ctx context.Context, ID uint64) (*UserPublicDTO, error)
	GetByMobile(ctx context.Context, mobile string) (*UserPublicDTO, error)
	GetByUsername(ctx context.Context, username string) (*UserPublicDTO, error)
	UpdateStatus(ctx context.Context, userID uint64, status string, actor AuditActorDTO) (*UserPublicDTO, error)
	GetAuthByID(ctx context.Context, ID uint64) (*UserAuthDTO, error)
	GetAuthByUsername(ctx context.Context, username string) (*UserAuthDTO, error)
	ListStaffRoles(ctx context.Context) ([]StaffRoleDTO, error)
	CreateStaffRole(ctx context.Context, role StaffRoleDTO) (*StaffRoleDTO, error)
	UpdateStaffRole(ctx context.Context, role StaffRoleDTO) (*StaffRoleDTO, error)
	DeleteStaffRole(ctx context.Context, roleName string) error
	GetUserRoleBinding(ctx context.Context, userID uint64) (*UserRoleBindingDTO, error)
	ReplaceUserRoleBinding(ctx context.Context, userID uint64, roleNames []string, actor AuditActorDTO) (*UserRoleBindingDTO, error)
	ListUserAuditLogs(ctx context.Context, userID uint64, filters UserAuditLogFilterDTO, opts metav1.ListMeta) (*UserAuditLogDTOList, error)
	CreateAdminAuditLog(ctx context.Context, log AdminAuditLogDTO) error
	ListAdminAuditLogs(ctx context.Context, filters AdminAuditLogFilterDTO, opts metav1.ListMeta) (*AdminAuditLogDTOList, error)
	RecordLogin(ctx context.Context, userID uint64, at time.Time) error
	CreateSession(ctx context.Context, session SessionDTO) (*SessionDTO, error)
	RefreshSession(ctx context.Context, sessionID string, currentHash, nextHash []byte, expiresAt time.Time) (*SessionDTO, error)
	RevokeSession(ctx context.Context, userID uint64, sessionID string) error
	RevokeAllSessions(ctx context.Context, userID uint64) error
	ValidateSession(ctx context.Context, userID uint64, sessionID string) (bool, error)
	ReplaceResourceScopes(ctx context.Context, userID uint64, scopes []ResourceScopeDTO) ([]ResourceScopeDTO, error)
}

type ResourceScopeDTO struct{ Domain, StoreID, TeamID string }
type resourceScopeStore interface {
	ReplaceResourceScopes(context.Context, uint64, []dv1.UserResourceScopeDO) error
}

func (u *userService) ReplaceResourceScopes(ctx context.Context, userID uint64, scopes []ResourceScopeDTO) ([]ResourceScopeDTO, error) {
	store, ok := u.userStore.(resourceScopeStore)
	if !ok {
		return nil, errors.WithCode(code2.ErrDatabase, "resource scope store is not configured")
	}
	models := make([]dv1.UserResourceScopeDO, 0, len(scopes))
	normalized := make([]ResourceScopeDTO, 0, len(scopes))
	seen := map[string]struct{}{}
	for _, scope := range scopes {
		scope.Domain = strings.ToLower(strings.TrimSpace(scope.Domain))
		scope.StoreID = strings.TrimSpace(scope.StoreID)
		scope.TeamID = strings.TrimSpace(scope.TeamID)
		if !authz.IsValidBusinessDomain(scope.Domain) {
			return nil, errors.WithCode(code2.ErrValidation, "resource scope domain is invalid")
		}
		if !authz.ResourceScopeMatchesDomain(authz.BusinessDomain(scope.Domain), scope.StoreID, scope.TeamID) {
			return nil, errors.WithCode(code2.ErrValidation, "resource scope does not match domain dimension")
		}
		key := scope.Domain + "\x00" + scope.StoreID + "\x00" + scope.TeamID
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		models = append(models, dv1.UserResourceScopeDO{UserID: int32(userID), Domain: scope.Domain, StoreID: scope.StoreID, TeamID: scope.TeamID, CreatedAt: time.Now().UTC()})
		normalized = append(normalized, scope)
	}
	if err := store.ReplaceResourceScopes(ctx, userID, models); err != nil {
		return nil, err
	}
	return normalized, nil
}

type SessionDTO struct {
	ID               string
	UserID           int32
	RefreshTokenHash []byte
	DeviceID         string
	DeviceName       string
	ExpiresAt        time.Time
}

type sessionStore interface {
	RecordLogin(ctx context.Context, id uint64, at time.Time) error
	CreateSession(ctx context.Context, session *dv1.UserSessionDO) error
	RotateSession(ctx context.Context, sessionID string, currentHash, nextHash []byte, expiresAt, usedAt time.Time) (*dv1.UserSessionDO, error)
	RevokeSession(ctx context.Context, userID uint64, sessionID string, at time.Time) error
	RevokeAllSessions(ctx context.Context, userID uint64, at time.Time) error
	SessionActive(ctx context.Context, userID uint64, sessionID string, at time.Time) (bool, error)
}

type userService struct {
	userStore dv1.UserStore
}

func (u *userService) RecordLogin(ctx context.Context, userID uint64, at time.Time) error {
	store, err := u.sessions()
	if err != nil {
		return err
	}
	return store.RecordLogin(ctx, userID, at)
}

func (u *userService) CreateSession(ctx context.Context, session SessionDTO) (*SessionDTO, error) {
	now := time.Now().UTC()
	model := &dv1.UserSessionDO{
		ID: uuid.NewString(), UserID: session.UserID,
		RefreshTokenHash: append([]byte(nil), session.RefreshTokenHash...),
		DeviceID:         session.DeviceID, DeviceName: session.DeviceName,
		CreatedAt: now, LastUsedAt: now, ExpiresAt: session.ExpiresAt.UTC(),
	}
	store, err := u.sessions()
	if err != nil {
		return nil, err
	}
	if err := store.CreateSession(ctx, model); err != nil {
		return nil, err
	}
	session.ID = model.ID
	return &session, nil
}

func (u *userService) RefreshSession(ctx context.Context, sessionID string, currentHash, nextHash []byte, expiresAt time.Time) (*SessionDTO, error) {
	store, err := u.sessions()
	if err != nil {
		return nil, err
	}
	model, err := store.RotateSession(ctx, sessionID, currentHash, nextHash, expiresAt, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	return &SessionDTO{ID: model.ID, UserID: model.UserID, DeviceID: model.DeviceID, DeviceName: model.DeviceName, ExpiresAt: model.ExpiresAt}, nil
}

func (u *userService) RevokeSession(ctx context.Context, userID uint64, sessionID string) error {
	store, err := u.sessions()
	if err != nil {
		return err
	}
	return store.RevokeSession(ctx, userID, sessionID, time.Now().UTC())
}

func (u *userService) RevokeAllSessions(ctx context.Context, userID uint64) error {
	store, err := u.sessions()
	if err != nil {
		return err
	}
	return store.RevokeAllSessions(ctx, userID, time.Now().UTC())
}

func (u *userService) ValidateSession(ctx context.Context, userID uint64, sessionID string) (bool, error) {
	store, err := u.sessions()
	if err != nil {
		return false, err
	}
	return store.SessionActive(ctx, userID, sessionID, time.Now().UTC())
}

func (u *userService) sessions() (sessionStore, error) {
	store, ok := u.userStore.(sessionStore)
	if !ok {
		return nil, errors.WithCode(code2.ErrDatabase, "session store is not configured")
	}
	return store, nil
}

var (
	usernamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]{2,31}$`)
	mobilePattern   = regexp.MustCompile(`^1([38][0-9]|14[579]|5[^4]|16[6]|7[1-35-8]|9[189])\d{8}$`)
	roleNamePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{2,63}$`)
)

const (
	minPasswordLength = 8
	maxPasswordBytes  = 72
)

func (u *userService) Create(ctx context.Context, user *UserDTO) error {
	if user == nil {
		return errors.WithCode(code2.ErrValidation, "用户信息不能为空")
	}
	if err := u.prepareNewUser(ctx, user, authz.LegacyUserRoleCustomer, string(authz.AccountStatusActive)); err != nil {
		return err
	}
	return u.userStore.Create(ctx, &user.UserDO)
}

func (u *userService) CreateStaff(ctx context.Context, user *UserDTO, roleNames []string, status string, actor AuditActorDTO) (*StaffUserDTO, error) {
	if user == nil {
		return nil, errors.WithCode(code2.ErrValidation, "用户信息不能为空")
	}
	roleNames = normalizeRoleNames(roleNames)
	if len(roleNames) == 0 {
		return nil, errors.WithCode(code2.ErrValidation, "staff roles are required")
	}
	if err := u.validateKnownRoleNames(ctx, roleNames); err != nil {
		return nil, err
	}

	normalizedStatus, err := normalizeMutableAccountStatus(status)
	if err != nil {
		return nil, err
	}
	if err = u.prepareNewUser(ctx, user, authz.LegacyUserRoleAdmin, string(normalizedStatus)); err != nil {
		return nil, err
	}

	authUser, err := u.userStore.CreateStaff(ctx, &user.UserDO, roleNames, newAuditActor(actor))
	if err != nil {
		return nil, err
	}
	return newStaffUserDTO(authUser)
}

func (u *userService) prepareNewUser(ctx context.Context, user *UserDTO, legacyRole authz.LegacyUserRole, defaultStatus string) error {
	if err := normalizeUserIdentifiers(user); err != nil {
		return err
	}
	if user.Role == 0 {
		user.Role = int(legacyRole)
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
		user.Status = defaultStatus
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
	return nil
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

func (u *userService) UpdateStatus(ctx context.Context, userID uint64, status string, actor AuditActorDTO) (*UserPublicDTO, error) {
	if userID == 0 {
		return nil, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}

	normalizedStatus, err := normalizeMutableAccountStatus(status)
	if err != nil {
		return nil, err
	}

	if err = u.userStore.UpdateStatus(ctx, userID, string(normalizedStatus), newAuditActor(actor)); err != nil {
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

	definitionsByRole := make(map[string]authz.RoleDefinition)
	for _, definition := range authz.BuiltinRoleDefinitions() {
		definitionsByRole[string(definition.Name)] = definition
	}

	result := make([]StaffRoleDTO, 0, len(roles))
	for _, role := range roles {
		definition, builtin := definitionsByRole[role.Name]
		domains := make([]string, 0, len(definition.Domains))
		for _, domain := range definition.Domains {
			domains = append(domains, string(domain))
		}
		sort.Strings(domains)
		result = append(result, StaffRoleDTO{
			Name:        role.Name,
			Description: role.Description,
			Permissions: append([]string(nil), role.Permissions...),
			Builtin:     builtin || role.Builtin,
			Domains:     coalesceDomains(role.Domains, domains),
		})
	}
	return result, nil
}

func (u *userService) CreateStaffRole(ctx context.Context, role StaffRoleDTO) (*StaffRoleDTO, error) {
	role.Name = strings.ToLower(strings.TrimSpace(role.Name))
	role.Description = strings.TrimSpace(role.Description)
	if role.Name == "" {
		return nil, errors.WithCode(code2.ErrValidation, "staff role name is required")
	}
	if authz.IsReservedNonStaffRoleName(role.Name) {
		return nil, errors.WithCode(code2.ErrValidation, "bootstrap and compatibility role names cannot be used as staff roles")
	}
	if authz.IsValidStaffRole(role.Name) {
		return nil, errors.WithCode(code2.ErrValidation, "built-in staff roles cannot be created again")
	}
	if !roleNamePattern.MatchString(role.Name) {
		return nil, errors.WithCode(code2.ErrValidation, "staff role name format is invalid")
	}
	if role.Description == "" {
		return nil, errors.WithCode(code2.ErrValidation, "staff role description is required")
	}

	permissions, err := normalizePermissionNames(role.Permissions)
	if err != nil {
		return nil, err
	}
	domains, err := normalizeBusinessDomains(role.Domains)
	if err != nil {
		return nil, err
	}

	createdRole, err := u.userStore.CreateRole(ctx, role.Name, role.Description, permissions, domains)
	if err != nil {
		return nil, err
	}
	return &StaffRoleDTO{
		Name:        createdRole.Name,
		Description: createdRole.Description,
		Permissions: append([]string(nil), createdRole.Permissions...),
		Builtin:     createdRole.Builtin,
		Domains:     append([]string(nil), createdRole.Domains...),
	}, nil
}

func (u *userService) UpdateStaffRole(ctx context.Context, role StaffRoleDTO) (*StaffRoleDTO, error) {
	role.Name = strings.ToLower(strings.TrimSpace(role.Name))
	role.Description = strings.TrimSpace(role.Description)
	if role.Name == "" {
		return nil, errors.WithCode(code2.ErrValidation, "staff role name is required")
	}
	if authz.IsReservedNonStaffRoleName(role.Name) {
		return nil, errors.WithCode(code2.ErrValidation, "bootstrap and compatibility role names cannot be used as staff roles")
	}
	if role.Description == "" {
		return nil, errors.WithCode(code2.ErrValidation, "staff role description is required")
	}
	permissions, err := normalizePermissionNames(role.Permissions)
	if err != nil {
		return nil, err
	}
	var domains []string
	if authz.IsValidStaffRole(role.Name) {
		definition, _ := builtinRoleDefinitionByName(role.Name)
		domains = domainsFromDefinition(definition)
	} else {
		domains, err = normalizeBusinessDomains(role.Domains)
		if err != nil {
			return nil, err
		}
	}

	updatedRole, err := u.userStore.UpdateRole(ctx, role.Name, role.Description, permissions, domains)
	if err != nil {
		return nil, err
	}
	return &StaffRoleDTO{
		Name:        updatedRole.Name,
		Description: updatedRole.Description,
		Permissions: append([]string(nil), updatedRole.Permissions...),
		Builtin:     updatedRole.Builtin,
		Domains:     append([]string(nil), updatedRole.Domains...),
	}, nil
}

func (u *userService) DeleteStaffRole(ctx context.Context, roleName string) error {
	roleName = strings.ToLower(strings.TrimSpace(roleName))
	if roleName == "" {
		return errors.WithCode(code2.ErrValidation, "staff role name is required")
	}
	if authz.IsReservedNonStaffRoleName(roleName) {
		return errors.WithCode(code2.ErrValidation, "bootstrap and compatibility role names cannot be used as staff roles")
	}
	if authz.IsValidStaffRole(roleName) {
		return errors.WithCode(code2.ErrValidation, "built-in staff roles cannot be deleted")
	}
	if !roleNamePattern.MatchString(roleName) {
		return errors.WithCode(code2.ErrValidation, "staff role name format is invalid")
	}
	return u.userStore.DeleteRole(ctx, roleName)
}

func (u *userService) GetUserRoleBinding(ctx context.Context, userID uint64) (*UserRoleBindingDTO, error) {
	authUser, err := u.userStore.GetAuthByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return newUserRoleBindingDTO(authUser)
}

func (u *userService) ReplaceUserRoleBinding(ctx context.Context, userID uint64, roleNames []string, actor AuditActorDTO) (*UserRoleBindingDTO, error) {
	roleNames = normalizeRoleNames(roleNames)
	if err := u.validateKnownRoleNames(ctx, roleNames); err != nil {
		return nil, err
	}
	authUser, err := u.userStore.ReplaceUserRoles(ctx, userID, roleNames, newAuditActor(actor))
	if err != nil {
		return nil, err
	}
	return newUserRoleBindingDTO(authUser)
}

func (u *userService) ListUserAuditLogs(ctx context.Context, userID uint64, filters UserAuditLogFilterDTO, opts metav1.ListMeta) (*UserAuditLogDTOList, error) {
	if userID == 0 {
		return nil, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}
	logs, err := u.userStore.ListAuditLogs(ctx, userID, dv1.UserAuditLogFilters{
		Action:             strings.TrimSpace(filters.Action),
		ActorUserID:        filters.ActorUserID,
		ActorPrincipalType: strings.TrimSpace(filters.ActorPrincipalType),
		CreatedAfter:       filters.CreatedAfter,
		CreatedBefore:      filters.CreatedBefore,
	}, opts)
	if err != nil {
		return nil, err
	}
	result := &UserAuditLogDTOList{TotalCount: logs.TotalCount}
	for _, item := range logs.Items {
		if item == nil {
			continue
		}
		result.Items = append(result.Items, &UserAuditLogDTO{
			ID:                 item.ID,
			UserID:             item.UserID,
			ActorUserID:        item.ActorUserID,
			ActorPrincipalType: item.ActorPrincipalType,
			Action:             item.Action,
			FromStatus:         item.FromStatus,
			ToStatus:           item.ToStatus,
			Detail:             item.Detail,
			CreatedAt:          item.CreatedAt,
		})
	}
	return result, nil
}

func (u *userService) CreateAdminAuditLog(ctx context.Context, log AdminAuditLogDTO) error {
	log.Action = strings.ToLower(strings.TrimSpace(log.Action))
	log.ActorPrincipalType = strings.TrimSpace(log.ActorPrincipalType)
	log.Detail = strings.TrimSpace(log.Detail)
	if !isValidAdminAuditAction(log.Action) {
		return errors.WithCode(code2.ErrValidation, "admin audit action is invalid")
	}
	if log.ActorPrincipalType == "" {
		return errors.WithCode(code2.ErrValidation, "admin audit actor principal type is required")
	}
	var correlationID *string
	if value := strings.TrimSpace(log.CorrelationID); value != "" {
		correlationID = &value
	}
	return u.userStore.CreateAdminAuditLog(ctx, &dv1.AdminAuditLogDO{
		TargetUserID:       log.TargetUserID,
		ActorUserID:        log.ActorUserID,
		ActorPrincipalType: log.ActorPrincipalType,
		Action:             log.Action,
		Detail:             log.Detail,
		CorrelationID:      correlationID, RequestID: strings.TrimSpace(log.RequestID), TargetType: strings.TrimSpace(log.TargetType), TargetID: strings.TrimSpace(log.TargetID), Domain: strings.TrimSpace(log.Domain), StoreID: strings.TrimSpace(log.StoreID), TeamID: strings.TrimSpace(log.TeamID),
	})
}

func (u *userService) ListAdminAuditLogs(ctx context.Context, filters AdminAuditLogFilterDTO, opts metav1.ListMeta) (*AdminAuditLogDTOList, error) {
	logs, err := u.userStore.ListAdminAuditLogs(ctx, dv1.AdminAuditLogFilters{
		TargetUserID:       filters.TargetUserID,
		Action:             strings.TrimSpace(filters.Action),
		ActorUserID:        filters.ActorUserID,
		ActorPrincipalType: strings.TrimSpace(filters.ActorPrincipalType),
		CreatedAfter:       filters.CreatedAfter,
		CreatedBefore:      filters.CreatedBefore,
	}, opts)
	if err != nil {
		return nil, err
	}

	result := &AdminAuditLogDTOList{TotalCount: logs.TotalCount}
	for _, item := range logs.Items {
		if item == nil {
			continue
		}
		result.Items = append(result.Items, &AdminAuditLogDTO{
			ID:                 item.ID,
			TargetUserID:       item.TargetUserID,
			ActorUserID:        item.ActorUserID,
			ActorPrincipalType: item.ActorPrincipalType,
			Action:             item.Action,
			Detail:             item.Detail,
			CorrelationID:      optionalAuditString(item.CorrelationID), RequestID: item.RequestID, TargetType: item.TargetType, TargetID: item.TargetID, Domain: item.Domain, StoreID: item.StoreID, TeamID: item.TeamID,
			CreatedAt: item.CreatedAt,
		})
	}
	return result, nil
}

func optionalAuditString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
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

func normalizeRoleNames(roleNames []string) []string {
	normalized := make([]string, 0, len(roleNames))
	seen := make(map[string]struct{}, len(roleNames))
	for _, roleName := range roleNames {
		value := strings.ToLower(strings.TrimSpace(roleName))
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	return normalized
}

func normalizeMutableAccountStatus(status string) (authz.AccountStatus, error) {
	normalizedStatus := authz.NormalizeAccountStatus(status)
	switch normalizedStatus {
	case authz.AccountStatusActive, authz.AccountStatusDisabled, authz.AccountStatusLocked:
		return normalizedStatus, nil
	default:
		return "", errors.WithCode(code2.ErrValidation, "account status must be one of: active, disabled, locked")
	}
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
		ID:             user.ID,
		Username:       user.Username,
		Mobile:         user.Mobile,
		Email:          user.Email,
		NickName:       user.NickName,
		Birthday:       birthday,
		Gender:         user.Gender,
		Status:         user.Status,
		MobileVerified: user.MobileVerified,
		EmailVerified:  user.EmailVerified,
		LastLoginAt:    user.LastLoginAt,
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
		UserPublicDTO:   *publicDTO,
		PasswordHash:    user.Password,
		LegacyRole:      int32(user.Role),
		StaffRoles:      append([]string(nil), user.StaffRoles...),
		Permissions:     append([]string(nil), user.Permissions...),
		ResourceDomains: append([]string(nil), user.ResourceDomains...),
		ResourceStores:  append([]string(nil), user.ResourceStores...),
		ResourceTeams:   append([]string(nil), user.ResourceTeams...),
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

func newStaffUserDTO(user *dv1.UserAuthDO) (*StaffUserDTO, error) {
	if user == nil {
		return nil, errors.WithCode(code.ErrUserNotFound, "用户不存在")
	}
	publicDTO, err := newUserPublicDTO(&user.UserDO)
	if err != nil {
		return nil, err
	}
	return &StaffUserDTO{
		User:        *publicDTO,
		StaffRoles:  append([]string(nil), user.StaffRoles...),
		Permissions: append([]string(nil), user.Permissions...),
	}, nil
}

func newAuditActor(actor AuditActorDTO) *dv1.AuditActor {
	if actor.UserID <= 0 && strings.TrimSpace(actor.PrincipalType) == "" {
		return nil
	}
	return &dv1.AuditActor{
		UserID:        actor.UserID,
		PrincipalType: strings.TrimSpace(actor.PrincipalType),
	}
}

func validateLegacyRole(role int32) error {
	if !authz.IsValidLegacyUserRole(role) {
		return errors.WithCode(code2.ErrValidation, "legacy role must be one of: 1(customer), 2(admin)")
	}
	return nil
}

func normalizePermissionNames(permissionNames []string) ([]string, error) {
	normalized := make(map[string]struct{}, len(permissionNames))
	for _, permissionName := range permissionNames {
		permissionName = strings.ToLower(strings.TrimSpace(permissionName))
		if permissionName == "" {
			continue
		}
		if !authz.IsValidPermission(permissionName) {
			return nil, errors.WithCode(code2.ErrValidation, "staff role permissions contain unknown values")
		}
		normalized[permissionName] = struct{}{}
	}
	if len(normalized) == 0 {
		return nil, errors.WithCode(code2.ErrValidation, "staff role permissions are required")
	}

	result := make([]string, 0, len(normalized))
	for permissionName := range normalized {
		result = append(result, permissionName)
	}
	sort.Strings(result)
	return result, nil
}

func normalizeBusinessDomains(domains []string) ([]string, error) {
	normalized := make(map[string]struct{}, len(domains))
	for _, domain := range domains {
		domain = strings.ToLower(strings.TrimSpace(domain))
		if domain == "" {
			continue
		}
		if !authz.IsValidBusinessDomain(domain) {
			return nil, errors.WithCode(code2.ErrValidation, "staff role domains contain unknown values")
		}
		normalized[domain] = struct{}{}
	}
	if len(normalized) == 0 {
		return nil, errors.WithCode(code2.ErrValidation, "staff role domains are required")
	}

	result := make([]string, 0, len(normalized))
	for domain := range normalized {
		result = append(result, domain)
	}
	sort.Strings(result)
	return result, nil
}

func (u *userService) validateKnownRoleNames(ctx context.Context, roleNames []string) error {
	if len(roleNames) == 0 {
		return nil
	}
	knownRoles, err := u.userStore.ListRoles(ctx)
	if err != nil {
		return err
	}
	known := make(map[string]struct{}, len(knownRoles))
	for _, role := range knownRoles {
		known[strings.ToLower(strings.TrimSpace(role.Name))] = struct{}{}
	}
	for _, roleName := range roleNames {
		if authz.IsReservedNonStaffRoleName(roleName) {
			return errors.WithCode(code2.ErrValidation, "staff roles contain reserved non-staff values")
		}
		if _, ok := known[roleName]; !ok {
			return errors.WithCode(code2.ErrValidation, "staff roles contain unknown values")
		}
	}
	return nil
}

func domainsFromDefinition(definition authz.RoleDefinition) []string {
	domains := make([]string, 0, len(definition.Domains))
	for _, domain := range definition.Domains {
		domains = append(domains, string(domain))
	}
	sort.Strings(domains)
	return domains
}

func coalesceDomains(primary, fallback []string) []string {
	if len(primary) > 0 {
		return append([]string(nil), primary...)
	}
	return append([]string(nil), fallback...)
}

func builtinRoleDefinitionByName(name string) (authz.RoleDefinition, bool) {
	for _, definition := range authz.BuiltinRoleDefinitions() {
		if string(definition.Name) == strings.ToLower(strings.TrimSpace(name)) {
			return definition, true
		}
	}
	return authz.RoleDefinition{}, false
}

func isValidAdminAuditAction(action string) bool {
	switch action {
	case dv1.AdminAuditActionStaffLoginSucceeded, dv1.AdminAuditActionBreakGlassSessionIssued,
		dv1.AdminAuditActionGoodsCreated, dv1.AdminAuditActionGoodsUpdated, dv1.AdminAuditActionGoodsDeleted,
		dv1.AdminAuditActionInventoryAdjusted, dv1.AdminAuditActionOrderClosed, dv1.AdminAuditActionOrderRefundRequested:
		return true
	default:
		return false
	}
}

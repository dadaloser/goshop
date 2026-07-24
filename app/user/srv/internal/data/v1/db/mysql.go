package db

import (
	"fmt"
	"goshop/app/pkg/authz"
	"goshop/app/pkg/code"
	appgorm "goshop/app/pkg/gorm"
	"goshop/app/pkg/options"
	dv1 "goshop/app/user/srv/internal/data/v1"
	errors2 "goshop/pkg/errors"
	"goshop/pkg/log"
	"strings"
	"sync"

	"gorm.io/gorm"

	"gorm.io/driver/mysql"
)

var (
	dbFactory *gorm.DB
	once      sync.Once
)

// 这个方法会返回gorm连接
// 还不够
// 这个方法应该返回的是全局的一个变量，如果一开始的时候没有初始化好，那么就初始化一次，后续呢直接拿到这个变量
func GetDBFactoryOr(mysqlOpts *options.MySQLOptions) (*gorm.DB, error) {
	if mysqlOpts == nil && dbFactory == nil {
		return nil, fmt.Errorf("failed to get mysql store fatory")
	}

	var err error
	once.Do(func() {
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
			mysqlOpts.Username,
			mysqlOpts.Password,
			mysqlOpts.Host,
			mysqlOpts.Port,
			mysqlOpts.Database)
		log.Infof("connecting mysql: host=%s port=%s database=%s user=%s",
			mysqlOpts.Host, mysqlOpts.Port, mysqlOpts.Database, mysqlOpts.Username)
		dbFactory, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err != nil {
			return
		}
		if err = dbFactory.Use(appgorm.NewResiliencePlugin(mysqlOpts.Resilience)); err != nil {
			dbFactory = nil
			return
		}

		sqlDB, dbErr := dbFactory.DB()
		if dbErr != nil {
			err = dbErr
			return
		}

		sqlDB.SetMaxOpenConns(mysqlOpts.MaxOpenConnections)
		sqlDB.SetMaxIdleConns(mysqlOpts.MaxIdleConnections)
		sqlDB.SetConnMaxLifetime(mysqlOpts.MaxConnectionLifetime)
		if mysqlOpts.AutoMigrate {
			if err = migrateUserSchema(dbFactory); err != nil {
				_ = sqlDB.Close()
				dbFactory = nil
				return
			}
		}
		if err = validateUserSchema(dbFactory); err != nil {
			_ = sqlDB.Close()
			dbFactory = nil
			return
		}
		if err = seedUserRBAC(dbFactory); err != nil {
			_ = sqlDB.Close()
			dbFactory = nil
			return
		}
		log.Infof("mysql connected: host=%s port=%s database=%s", mysqlOpts.Host, mysqlOpts.Port, mysqlOpts.Database)
	})

	if dbFactory == nil || err != nil {
		return nil, errors2.WrapC(err, code.ErrConnectDB, "failed to get mysql store factory")
	}
	return dbFactory, nil
}

func migrateUserSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("user schema migration failed: nil db")
	}
	if err := db.AutoMigrate(&dv1.UserDO{}, &dv1.UserSessionDO{}, &dv1.VerificationCodeDO{}, &dv1.UserResourceScopeDO{}, &dv1.RoleDO{}, &dv1.UserRoleDO{}, &dv1.RolePermissionDO{}, &dv1.RoleDomainDO{}, &dv1.UserAuditLogDO{}, &dv1.AdminAuditLogDO{}); err != nil {
		return fmt.Errorf("user schema migration failed: %w", err)
	}
	return nil
}

func validateUserSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("user schema validation failed: nil db")
	}
	for _, table := range userSchemaChecks() {
		if !db.Migrator().HasTable(table.model) {
			return fmt.Errorf("user schema validation failed: required table %q does not exist", table.model.TableName())
		}
		for _, column := range table.required {
			if !db.Migrator().HasColumn(table.model, column) {
				return fmt.Errorf("user schema validation failed: required column %q.%q does not exist", table.model.TableName(), column)
			}
		}
	}
	return nil
}

type schemaTableCheck struct {
	model    interface{ TableName() string }
	required []string
}

func userSchemaChecks() []schemaTableCheck {
	return []schemaTableCheck{
		{
			model: &dv1.UserDO{},
			required: []string{
				"id",
				"add_time",
				"update_time",
				"deleted_at",
				"is_deleted",
				"username",
				"mobile",
				"email",
				"password",
				"nick_name",
				"birthday",
				"gender",
				"role",
				"account_status",
				"mobile_verified",
				"email_verified",
				"last_login_at",
			},
		},
		{
			model:    &dv1.UserSessionDO{},
			required: []string{"id", "user_id", "refresh_token_hash", "device_id", "device_name", "created_at", "last_used_at", "expires_at", "revoked_at"},
		},
		{
			model:    &dv1.VerificationCodeDO{},
			required: []string{"id", "channel", "purpose", "destination_hash", "code_hash", "attempts", "expires_at", "consumed_at", "created_at"},
		},
		{
			model:    &dv1.UserResourceScopeDO{},
			required: []string{"id", "user_id", "domain", "store_id", "team_id", "created_at"},
		},
		{
			model:    &dv1.RoleDO{},
			required: []string{"id", "name", "description"},
		},
		{
			model:    &dv1.UserRoleDO{},
			required: []string{"id", "user_id", "role_id"},
		},
		{
			model:    &dv1.RolePermissionDO{},
			required: []string{"id", "role_id", "permission"},
		},
		{
			model:    &dv1.RoleDomainDO{},
			required: []string{"id", "role_id", "domain"},
		},
		{
			model:    &dv1.UserAuditLogDO{},
			required: []string{"id", "user_id", "actor_user_id", "actor_principal_type", "action", "from_status", "to_status", "detail", "add_time"},
		},
		{
			model:    &dv1.AdminAuditLogDO{},
			required: []string{"id", "target_user_id", "actor_user_id", "actor_principal_type", "action", "detail", "correlation_id", "request_id", "target_type", "target_id", "domain", "store_id", "team_id", "add_time"},
		},
	}
}

func seedUserRBAC(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("seed user RBAC failed: nil db")
	}

	for _, role := range authz.BuiltinRoleDefinitions() {
		record := dv1.RoleDO{
			Name:        string(role.Name),
			Description: role.Description,
		}
		if err := db.Where("name = ?", record.Name).Assign(map[string]interface{}{
			"description": record.Description,
		}).FirstOrCreate(&record).Error; err != nil {
			return fmt.Errorf("seed user RBAC role %q failed: %w", role.Name, err)
		}

		for _, permission := range role.Permissions {
			permissionValue := strings.TrimSpace(string(permission))
			if permissionValue == "" {
				continue
			}
			rolePermission := dv1.RolePermissionDO{
				RoleID:     record.ID,
				Permission: permissionValue,
			}
			if err := db.Where("role_id = ? AND permission = ?", rolePermission.RoleID, rolePermission.Permission).
				FirstOrCreate(&rolePermission).Error; err != nil {
				return fmt.Errorf("seed user RBAC permission %q for role %q failed: %w", permission, role.Name, err)
			}
		}
		for _, domain := range role.Domains {
			domainValue := strings.TrimSpace(string(domain))
			if domainValue == "" {
				continue
			}
			roleDomain := dv1.RoleDomainDO{
				RoleID: record.ID,
				Domain: domainValue,
			}
			if err := db.Where("role_id = ? AND domain = ?", roleDomain.RoleID, roleDomain.Domain).
				FirstOrCreate(&roleDomain).Error; err != nil {
				return fmt.Errorf("seed user RBAC domain %q for role %q failed: %w", domain, role.Name, err)
			}
		}
	}
	return nil
}

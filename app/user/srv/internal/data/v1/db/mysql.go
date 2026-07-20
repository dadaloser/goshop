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
	if err := db.AutoMigrate(&dv1.UserDO{}, &dv1.RoleDO{}, &dv1.UserRoleDO{}, &dv1.RolePermissionDO{}); err != nil {
		return fmt.Errorf("user schema migration failed: %w", err)
	}
	return nil
}

func validateUserSchema(db *gorm.DB) error {
	if db == nil {
		return fmt.Errorf("user schema validation failed: nil db")
	}
	if !db.Migrator().HasTable(&dv1.UserDO{}) {
		return fmt.Errorf("user schema validation failed: required table %q does not exist", (&dv1.UserDO{}).TableName())
	}

	requiredColumns := []string{
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
	}
	for _, column := range requiredColumns {
		if !db.Migrator().HasColumn(&dv1.UserDO{}, column) {
			return fmt.Errorf("user schema validation failed: required column %q.%q does not exist", (&dv1.UserDO{}).TableName(), column)
		}
	}
	rbacTables := []struct {
		model   interface{ TableName() string }
		columns []string
	}{
		{
			model:   &dv1.RoleDO{},
			columns: []string{"id", "name", "description"},
		},
		{
			model:   &dv1.UserRoleDO{},
			columns: []string{"id", "user_id", "role_id"},
		},
		{
			model:   &dv1.RolePermissionDO{},
			columns: []string{"id", "role_id", "permission"},
		},
	}
	for _, table := range rbacTables {
		if !db.Migrator().HasTable(table.model) {
			return fmt.Errorf("user schema validation failed: required table %q does not exist", table.model.TableName())
		}
		for _, column := range table.columns {
			if !db.Migrator().HasColumn(table.model, column) {
				return fmt.Errorf("user schema validation failed: required column %q.%q does not exist", table.model.TableName(), column)
			}
		}
	}
	return nil
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
	}
	return nil
}

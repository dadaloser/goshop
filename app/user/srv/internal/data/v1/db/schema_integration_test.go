package db

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"goshop/app/pkg/options"
)

const (
	schemaTestMySQLDSNEnv      = "GOSHOP_USER_SCHEMA_TEST_MYSQL_DSN"
	schemaTestMySQLUsernameEnv = "GOSHOP_SCHEMA_TEST_MYSQL_USERNAME"
	schemaTestMySQLPasswordEnv = "GOSHOP_SCHEMA_TEST_MYSQL_PASSWORD"
	schemaTestMySQLHostEnv     = "GOSHOP_SCHEMA_TEST_MYSQL_HOST"
	schemaTestMySQLPortEnv     = "GOSHOP_SCHEMA_TEST_MYSQL_PORT"
	schemaTestMySQLDatabaseEnv = "GOSHOP_USER_SCHEMA_TEST_MYSQL_DATABASE"

	serviceMySQLUsernameEnv = "USER_MYSQL_USERNAME"
	serviceMySQLPasswordEnv = "USER_MYSQL_PASSWORD"
	serviceMySQLHostEnv     = "USER_MYSQL_HOST"
	serviceMySQLPortEnv     = "USER_MYSQL_PORT"
	serviceMySQLDatabaseEnv = "USER_MYSQL_DATABASE"
)

func TestUserStartupValidationRealDB(t *testing.T) {
	db, dsn := mustOpenSchemaIntegrationDB(t)
	resetUserFactory()
	t.Cleanup(resetUserFactory)

	prepareUserSchemaMigrations(t, db)

	mysqlOpts := mustSchemaTestMySQLOptions(t, dsn)
	if _, err := GetDBFactoryOr(mysqlOpts); err != nil {
		t.Fatalf("GetDBFactoryOr(schema migrated db) error = %v", err)
	}
}

func mustOpenSchemaIntegrationDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()

	dsn := schemaIntegrationDSNFromEnv()
	if dsn == "" {
		t.Skipf("set %s, %s/%s[/HOST/PORT/DATABASE], or %s/%s[/HOST/PORT/DATABASE] to run real MySQL schema integration tests",
			schemaTestMySQLDSNEnv,
			schemaTestMySQLUsernameEnv, schemaTestMySQLPasswordEnv,
			serviceMySQLUsernameEnv, serviceMySQLPasswordEnv)
	}

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN(%s) error = %v", schemaTestMySQLDSNEnv, err)
	}
	cfg.MultiStatements = true

	db, err := gorm.Open(gormmysql.Open(cfg.FormatDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatalf("gorm.Open(schema test db) error = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB(schema test db) error = %v", err)
	}
	sqlDB.SetMaxOpenConns(32)
	sqlDB.SetMaxIdleConns(8)
	sqlDB.SetConnMaxLifetime(time.Minute)
	t.Cleanup(func() {
		_ = sqlDB.Close()
	})

	return db, dsn
}

func schemaIntegrationDSNFromEnv() string {
	if dsn := strings.TrimSpace(os.Getenv(schemaTestMySQLDSNEnv)); dsn != "" {
		return dsn
	}

	username := strings.TrimSpace(os.Getenv(schemaTestMySQLUsernameEnv))
	password := strings.TrimSpace(os.Getenv(schemaTestMySQLPasswordEnv))
	if username == "" || password == "" {
		username = strings.TrimSpace(os.Getenv(serviceMySQLUsernameEnv))
		password = strings.TrimSpace(os.Getenv(serviceMySQLPasswordEnv))
		if username == "" || password == "" {
			return ""
		}
	}

	host := strings.TrimSpace(os.Getenv(schemaTestMySQLHostEnv))
	if host == "" {
		host = strings.TrimSpace(os.Getenv(serviceMySQLHostEnv))
		if host == "" {
			host = "127.0.0.1"
		}
	}
	port := strings.TrimSpace(os.Getenv(schemaTestMySQLPortEnv))
	if port == "" {
		port = strings.TrimSpace(os.Getenv(serviceMySQLPortEnv))
		if port == "" {
			port = "3306"
		}
	}
	database := strings.TrimSpace(os.Getenv(schemaTestMySQLDatabaseEnv))
	if database == "" {
		database = strings.TrimSpace(os.Getenv(serviceMySQLDatabaseEnv))
		if database == "" {
			database = "goshop_user_srv"
		}
	}

	cfg := mysqldriver.Config{
		User:                 username,
		Passwd:               password,
		Net:                  "tcp",
		Addr:                 net.JoinHostPort(host, port),
		DBName:               database,
		ParseTime:            true,
		Loc:                  time.Local,
		AllowNativePasswords: true,
	}
	return cfg.FormatDSN()
}

func prepareUserSchemaMigrations(t *testing.T, db *gorm.DB) {
	t.Helper()

	dropStatements := []string{
		"DROP TABLE IF EXISTS `user_resource_scopes`",
		"DROP TABLE IF EXISTS `verification_codes`",
		"DROP TABLE IF EXISTS `user_sessions`",
		"DROP TABLE IF EXISTS `admin_audit_logs`",
		"DROP TABLE IF EXISTS `user_audit_logs`",
		"DROP TABLE IF EXISTS `role_domains`",
		"DROP TABLE IF EXISTS `role_permissions`",
		"DROP TABLE IF EXISTS `user_roles`",
		"DROP TABLE IF EXISTS `roles`",
		"DROP TABLE IF EXISTS `user`",
	}
	for _, statement := range dropStatements {
		if err := db.Exec(statement).Error; err != nil {
			t.Fatalf("cleanup schema test tables error = %v", err)
		}
	}
	t.Cleanup(func() {
		for _, statement := range dropStatements {
			_ = db.Exec(statement).Error
		}
	})

	for _, path := range userMigrationFiles(t) {
		applyMigrationFile(t, db, path)
	}
	applySelectedMigrationStatements(t, db,
		userSharedMigrationPath(t),
		func(statement string) bool {
			return strings.Contains(statement, "CREATE TABLE `user_resource_scopes`") ||
				strings.Contains(statement, "ALTER TABLE `admin_audit_logs`")
		},
	)
}

func userMigrationFiles(t *testing.T) []string {
	t.Helper()

	root := migrationRoot(t)
	return []string{
		filepath.Join(root, "migrations/202607040001_user_create_core_tables.up.sql"),
		filepath.Join(root, "migrations/202607050001_user_add_identity_columns.up.sql"),
		filepath.Join(root, "migrations/202607200001_user_add_audit_logs.up.sql"),
		filepath.Join(root, "migrations/202607210001_user_add_admin_audit_logs.up.sql"),
		filepath.Join(root, "migrations/202607220001_user_add_account_status.up.sql"),
		filepath.Join(root, "migrations/202607220002_user_add_rbac_core_tables.up.sql"),
		filepath.Join(root, "migrations/202607230001_user_add_identity_and_sessions.up.sql"),
		filepath.Join(root, "migrations/202607230006_user_add_review_rbac.up.sql"),
	}
}

func userSharedMigrationPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(migrationRoot(t), "migrations/202607230002_admin_resource_scopes_and_refunds.up.sql")
}

func migrationRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../../../../"))
}

func applyMigrationFile(t *testing.T, db *gorm.DB, path string) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}
	if err = db.Exec(string(content)).Error; err != nil {
		t.Fatalf("apply migration %s error = %v", filepath.Base(path), err)
	}
}

func applySelectedMigrationStatements(t *testing.T, db *gorm.DB, path string, keep func(statement string) bool) {
	t.Helper()

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", path, err)
	}

	for _, statement := range strings.Split(string(content), ";") {
		statement = strings.TrimSpace(statement)
		if statement == "" || !keep(statement) {
			continue
		}
		if err = db.Exec(statement + ";").Error; err != nil {
			t.Fatalf("apply filtered migration from %s error = %v\nstatement:\n%s", filepath.Base(path), err, statement)
		}
	}
}

func mustSchemaTestMySQLOptions(t *testing.T, dsn string) *options.MySQLOptions {
	t.Helper()

	cfg, err := mysqldriver.ParseDSN(dsn)
	if err != nil {
		t.Fatalf("mysql.ParseDSN(schema test dsn) error = %v", err)
	}
	host, port, err := splitMySQLAddr(cfg.Addr)
	if err != nil {
		t.Fatalf("splitMySQLAddr(%q) error = %v", cfg.Addr, err)
	}
	return &options.MySQLOptions{
		Host:                  host,
		Port:                  port,
		Username:              cfg.User,
		Password:              cfg.Passwd,
		Database:              cfg.DBName,
		MaxIdleConnections:    8,
		MaxOpenConnections:    32,
		MaxConnectionLifetime: time.Minute,
		LogLevel:              1,
	}
}

func splitMySQLAddr(addr string) (string, string, error) {
	if addr == "" {
		return "127.0.0.1", "3306", nil
	}
	host, port, err := net.SplitHostPort(addr)
	if err == nil {
		return host, port, nil
	}
	if strings.Count(addr, ":") == 0 {
		return addr, "3306", nil
	}
	return "", "", err
}

func resetUserFactory() {
	dbFactory = nil
	once = sync.Once{}
}

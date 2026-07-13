package gorm

import (
	"context"
	"errors"
	"testing"

	"goshop/gmicro/resilience"

	mysqlDriver "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	gormio "gorm.io/gorm"
)

func TestResiliencePluginRegistersCallbacks(t *testing.T) {
	db, err := gormio.Open(gormmysql.New(gormmysql.Config{
		DSN:                       "user:password@tcp(127.0.0.1:3306)/test",
		SkipInitializeWithVersion: true,
	}), &gormio.Config{
		DisableAutomaticPing: true,
		DryRun:               true,
	})
	if err != nil {
		t.Fatalf("gorm.Open() error = %v", err)
	}
	if err := db.Use(NewResiliencePlugin(resilience.NewOptions())); err != nil {
		t.Fatalf("Use() error = %v", err)
	}

	var destination []struct{ ID int }
	if err := db.WithContext(t.Context()).Find(&destination).Error; err != nil {
		t.Fatalf("Find() error = %v", err)
	}
}

func TestIsMySQLDependencyError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "connection error", err: errors.New("connection refused"), want: true},
		{name: "deadline", err: context.DeadlineExceeded, want: true},
		{name: "not found", err: gormio.ErrRecordNotFound, want: false},
		{name: "duplicate", err: &mysqlDriver.MySQLError{Number: 1062, Message: "duplicate"}, want: false},
		{name: "foreign key", err: &mysqlDriver.MySQLError{Number: 1452, Message: "constraint"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isMySQLDependencyError(tt.err); got != tt.want {
				t.Fatalf("isMySQLDependencyError() = %v, want %v", got, tt.want)
			}
		})
	}
}

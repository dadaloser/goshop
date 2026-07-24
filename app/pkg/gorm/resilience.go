package gorm

import (
	"context"
	"errors"

	"goshop/gmicro/resilience"

	mysqlDriver "github.com/go-sql-driver/mysql"
	gormio "gorm.io/gorm"
)

const resilienceCallKey = "goshop:resilience:mysql:call"

// ResiliencePlugin applies Sentinel timeout, isolation, and circuit breaking to GORM operations.
type ResiliencePlugin struct {
	options *resilience.Options
	guard   *resilience.Guard
}

// NewResiliencePlugin creates a GORM plugin using the supplied dependency policy.
func NewResiliencePlugin(options *resilience.Options) *ResiliencePlugin {
	return &ResiliencePlugin{options: options}
}

// Name returns the stable GORM plugin identifier.
func (p *ResiliencePlugin) Name() string {
	return "goshop:dependency_resilience"
}

// Initialize registers callbacks around all GORM operation processors.
func (p *ResiliencePlugin) Initialize(db *gormio.DB) error {
	guard, err := resilience.NewGuard("mysql", p.options, isMySQLDependencyError)
	if err != nil {
		return err
	}
	p.guard = guard

	if err := db.Callback().Create().Before("*").Register(
		p.beforeName("create"),
		p.before("create"),
	); err != nil {
		return err
	}
	if err := db.Callback().Create().After("*").Register(
		p.afterName("create"),
		p.after,
	); err != nil {
		return err
	}
	if err := db.Callback().Query().Before("*").Register(
		p.beforeName("query"),
		p.before("query"),
	); err != nil {
		return err
	}
	if err := db.Callback().Query().After("*").Register(
		p.afterName("query"),
		p.after,
	); err != nil {
		return err
	}
	if err := db.Callback().Update().Before("*").Register(
		p.beforeName("update"),
		p.before("update"),
	); err != nil {
		return err
	}
	if err := db.Callback().Update().After("*").Register(
		p.afterName("update"),
		p.after,
	); err != nil {
		return err
	}
	if err := db.Callback().Delete().Before("*").Register(
		p.beforeName("delete"),
		p.before("delete"),
	); err != nil {
		return err
	}
	if err := db.Callback().Delete().After("*").Register(
		p.afterName("delete"),
		p.after,
	); err != nil {
		return err
	}
	if err := db.Callback().Raw().Before("*").Register(
		p.beforeName("raw"),
		p.before("raw"),
	); err != nil {
		return err
	}
	return db.Callback().Raw().After("*").Register(
		p.afterName("raw"),
		p.after,
	)
}

func (p *ResiliencePlugin) before(resource string) func(*gormio.DB) {
	return func(db *gormio.DB) {
		call, err := p.guard.Start(db.Statement.Context, resource)
		if err != nil {
			db.Error = db.AddError(err)
			return
		}
		db.Statement.Context = call.Context()
		db.Statement.Settings.Store(resilienceCallKey, call)
	}
}

func (p *ResiliencePlugin) after(db *gormio.DB) {
	value, ok := db.Statement.Settings.LoadAndDelete(resilienceCallKey)
	if !ok {
		return
	}
	call, ok := value.(*resilience.Call)
	if !ok {
		return
	}
	call.Finish(db.Error)
}

func (p *ResiliencePlugin) beforeName(resource string) string {
	return p.Name() + ":before_" + resource
}

func (p *ResiliencePlugin) afterName(resource string) string {
	return p.Name() + ":after_" + resource
}

func isMySQLDependencyError(err error) bool {
	if err == nil || errors.Is(err, context.Canceled) {
		return false
	}
	ignored := []error{
		gormio.ErrRecordNotFound,
		gormio.ErrInvalidData,
		gormio.ErrInvalidValue,
		gormio.ErrMissingWhereClause,
		gormio.ErrModelValueRequired,
		gormio.ErrPrimaryKeyRequired,
		gormio.ErrUnsupportedRelation,
	}
	for _, ignoredErr := range ignored {
		if errors.Is(err, ignoredErr) {
			return false
		}
	}

	var mysqlErr *mysqlDriver.MySQLError
	if errors.As(err, &mysqlErr) {
		switch mysqlErr.Number {
		case 1062, 1451, 1452:
			return false
		}
	}
	return true
}

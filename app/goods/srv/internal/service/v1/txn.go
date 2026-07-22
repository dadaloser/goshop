package v1

import (
	"context"
	"fmt"

	"gorm.io/gorm"
)

type txExecutor interface {
	DB() *gorm.DB
	Commit() error
	Rollback() error
}

type gormTxn struct {
	db *gorm.DB
}

func (t gormTxn) DB() *gorm.DB {
	return t.db
}

func (t gormTxn) Commit() error {
	return t.db.Commit().Error
}

func (t gormTxn) Rollback() error {
	return t.db.Rollback().Error
}

func withTxnExecutor(ctx context.Context, txn txExecutor, action string, fn func(txExecutor) error) (err error) {
	if txn == nil {
		return fmt.Errorf("%s transaction is nil", action)
	}
	committed := false
	defer func() {
		if panicValue := recover(); panicValue != nil {
			if !committed {
				_ = txn.Rollback()
			}
			err = fmt.Errorf("%s transaction panic: %v", action, panicValue)
			return
		}
		if !committed {
			_ = txn.Rollback()
		}
	}()

	if err := fn(txn); err != nil {
		return err
	}
	if err := txn.Commit(); err != nil {
		return fmt.Errorf("commit %s transaction: %w", action, err)
	}
	committed = true
	return nil
}

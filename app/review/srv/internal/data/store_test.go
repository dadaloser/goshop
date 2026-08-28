package data

import (
	stderrors "errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestTranslateClassifiesStructuredDatabaseErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "wrapped duplicate key",
			err:  fmt.Errorf("create review: %w", &mysql.MySQLError{Number: 1062}),
			want: ErrConflict,
		},
		{
			name: "record not found",
			err:  gorm.ErrRecordNotFound,
			want: ErrNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := translate(tt.err); !stderrors.Is(got, tt.want) {
				t.Errorf("translate(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestTranslateDoesNotClassifyTextOnlyUniqueErrorAsConflict(t *testing.T) {
	err := stderrors.New("unique value cannot be normalized")
	if got := translate(err); got != err {
		t.Errorf("translate(%v) = %v, want original error", err, got)
	}
}

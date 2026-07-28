package v1

import (
	"os"
	"testing"

	"goshop/app/pkg/errorcatalog"
)

func TestMain(m *testing.M) {
	errorcatalog.RegisterAll()
	os.Exit(m.Run())
}

package v1

import (
	"goshop/app/pkg/errorcatalog"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	errorcatalog.RegisterAll()
	os.Exit(m.Run())
}

package main

import (
	"fmt"
	"goshop/app/goshop/admin"
	"os"
	"runtime"
)

// --config=./configs/api/api.yaml
func main() {
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	if err := admin.NewApp("admin-server").Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

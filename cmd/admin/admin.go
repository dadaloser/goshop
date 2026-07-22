package main

import (
	"goshop/app/goshop/admin"
	"os"
	"runtime"
)

// --config=./configs/api/api.yaml
func main() {
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	admin.NewApp("admin-server").Run()
}

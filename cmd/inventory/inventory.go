package main

import (
	"goshop/app/inventory/srv"
	"math/rand"
	"os"
	"runtime"
	"time"
)

// 程序实参: --config=./configs/inventory/srv.yaml
func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	srv.NewApp("inventory-server").Run()
}

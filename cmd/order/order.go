package main

import (
	"goshop/app/order/srv"
	"math/rand"
	"os"
	"runtime"
	"time"
)

// 程序实参: --config=./configs/order/srv.yaml
func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	srv.NewApp("order-server").Run()
}

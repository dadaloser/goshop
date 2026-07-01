package main

import (
	"goshop/app/user/srv"
	"math/rand"
	"os"
	"runtime"
	"time"
)

// 程序实参: --config=./configs/user/srv.yaml
func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	srv.NewApp("user-server").Run()
}

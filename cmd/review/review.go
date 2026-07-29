package main

import (
	"math/rand"
	"os"
	"runtime"
	"time"

	"goshop/app/review/srv"
)

// 程序实参: --config=./configs/review/srv.yaml
func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	srv.NewApp("review-server").Run()
}

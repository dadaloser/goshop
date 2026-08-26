package main

import (
	"fmt"
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
	if err := srv.NewApp("review-server").Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

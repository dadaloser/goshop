package main

import (
	"fmt"
	"goshop/app/goshop/api"
	"math/rand"
	"os"
	"runtime"
	"time"
)

// 程序实参: --config=./configs/api/api.yaml
func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	if err := api.NewApp("api-server").Run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

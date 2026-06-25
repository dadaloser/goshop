package main

import (
	"goshop/app/goshop/admin"
	"math/rand"
	"os"
	"runtime"
	"time"
)

func main() {
	rand.New(rand.NewSource(time.Now().UnixNano()))
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}
	admin.NewApp("admin-server").Run()
}

package helpers

import (
	"log"
	"os"
	"reflect"
	"runtime"
)

func GetTestName(test func() error) string {
	return runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name()
}

func RunMinter(isReady chan bool) {
	logger := log.New(os.Stdout, "minter-node ", log.LstdFlags)

	logger.Println("Starting Minter Node...")

	logger.Println("Minter Node ready")
	isReady <- true
}

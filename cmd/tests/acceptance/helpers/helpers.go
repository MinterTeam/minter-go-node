package helpers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strings"
	"time"
)

var (
	path string
)

func init() {
	path, _ = os.Getwd()
}

func GetTestName(test func() error) string {
	path := strings.Split(runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name(), "/")
	return path[len(path)-1]
}

func CopyFile(fromFile string, toFile string) {
	from, err := os.Open(fromFile)
	if err != nil {
		log.Fatal(err)
	}
	defer from.Close()

	to, err := os.OpenFile(toFile, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		log.Fatal(err)
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		log.Fatal(err)
	}
}

func RunMinter(isReady chan bool) {
	StopMinter()

	logger := log.New(os.Stdout, "minter-node ", log.LstdFlags)

	logger.Println("Starting Minter Node...")

	err := os.MkdirAll(path+"/docker/data/.tendermint/config/", os.ModePerm)

	if err != nil {
		panic(err)
	}

	cmd := exec.Command("cp", "-r", path+"/docker/default/", path+"/docker/data/.tendermint/config/")
	cmd.Start()
	cmd.Wait()

	cmd = exec.Command("docker-compose", "--file", path+"/docker/docker-compose.yml", "--project-name", "minter-test", "up")
	cmd.Stderr = os.Stderr
	err = cmd.Start()
	if err != nil {
		logger.Fatal(err)
	}

	timer := time.NewTimer(20 * time.Second)

	go func() {
		<-timer.C
		StopMinter()
		logger.Fatalln("Minter node start timeout...")
	}()

	for {
		_, err = http.Get("http://localhost:8841/api/status")

		if err == nil {
			isReady <- true
			return
		}

		time.Sleep(1 * time.Second)
	}

	err = cmd.Wait()
	if err != nil {
		logger.Fatalf("Minter finished with error: %v", err)
	}
}

func StopMinter() {
	fmt.Println("Stopping Minter...")
	cmd := exec.Command("docker-compose", "--file", path+"/docker/docker-compose.yml", "--project-name", "minter-test", "stop")
	cmd.Start()
	cmd.Wait()

	cmd = exec.Command("rm", "-rf", path+"/docker/data/.minter")
	cmd.Start()
	err := cmd.Wait()

	if err != nil {
		log.Fatalf("%s", err)
	}

	cmd = exec.Command("rm", "-rf", path+"/docker/data/.tendermint")
	cmd.Start()
	err = cmd.Wait()

	if err != nil {
		log.Fatalf("%s", err)
	}
}

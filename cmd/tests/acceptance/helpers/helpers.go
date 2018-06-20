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
	"time"
)

var (
	path string
)

func init() {
	path, _ = os.Getwd()
}

func GetTestName(test func() error) string {
	return runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name()
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
	logger := log.New(os.Stdout, "minter-node ", log.LstdFlags)

	logger.Println("Starting Minter Node...")

	cmd := exec.Command("mkdir", "-p", path+"/docker/data/.tendermint/config/")
	cmd.Start()
	cmd.Wait()

	cmd = exec.Command("cp", "-r", path+"/docker/default/", path+"/docker/data/.tendermint/config/")
	cmd.Start()
	cmd.Wait()

	cmd = exec.Command("docker-compose", "--file", path+"/docker/docker-compose.yml", "--project-name", "minter-test", "start")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err := cmd.Start()
	if err != nil {
		logger.Fatal(err)
	}

	err = cmd.Wait()
	if err != nil {
		logger.Fatalf("Minter finished with error: %v", err)
	}

	timer := time.NewTimer(20 * time.Second)

	go func() {
		<-timer.C
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
}

func StopMinter() {
	fmt.Println("Stopping Minter...")
	cmd := exec.Command("docker-compose", "--file", path+"/docker/docker-compose.yml", "--project-name", "minter-test", "stop")
	cmd.Start()
	cmd.Wait()

	err := os.RemoveAll(path + "/docker/data/.minter")
	err = os.RemoveAll(path + "/docker/data/.tendermint")

	if err != nil {
		fmt.Printf("%s", err)
	}
}

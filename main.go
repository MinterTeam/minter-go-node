package main

import (
	"os"
	"os/exec"
	"github.com/tendermint/tmlibs/log"
	"minter/core/minter"
	"github.com/tendermint/abci/server"
	"minter/api"
	"github.com/tendermint/tmlibs/common"
	"fmt"
	"minter/helpers"
	"time"
)

func main() {
	logger := log.NewTMLogger(log.NewSyncWriter(os.Stdout))

	app := minter.NewMinterBlockchain()

	// Start the listener
	srv, err := server.NewServer("tcp://0.0.0.0:46658", "socket", app)
	if err != nil {
		panic(err)
	}
	srv.SetLogger(logger.With("module", "abci-server"))
	if err := srv.Start(); err != nil {
		panic(err)
	}

	go runTendermint()
	go api.RunApi(app)

	// Wait forever
	common.TrapSignal(func() {
		// Cleanup
		app.Stop()
		srv.Stop()
	})

}

func runTendermint() {

	wd, _ := os.Getwd()
	homeDir := wd + "/.data/tendermint"

	if _, err := os.Stat(homeDir); os.IsNotExist(err) {
		init := exec.Command("tendermint", "init", "--home", homeDir)

		init.Start()
		init.Wait()

		helpers.Copy(wd+"/default_config/genesis.json", homeDir+"/config/genesis.json")
		helpers.Copy(wd+"/default_config/config.toml", homeDir+"/config/config.toml")
	}

	tendermintLog, _ := os.OpenFile(wd + "/logs/tendermint.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer tendermintLog.Close()

	tendermintErrorLog, _ := os.OpenFile(wd + "/logs/tendermint.err.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer tendermintErrorLog.Close()

	for {
		tendermintInstance := exec.Command("tendermint", "node", "--home", homeDir)
		tendermintInstance.Stdout = tendermintLog
		tendermintInstance.Stderr = tendermintErrorLog
		err := tendermintInstance.Start()

		if err != nil {
			panic(err)
		}

		err = tendermintInstance.Wait()
		tendermintErrorLog.WriteString("Tendermint exit status: " + err.Error() + "\n")
		fmt.Printf("Restaring tendermint after 5 secs\n")
		time.Sleep(5 * time.Second)
	}
}

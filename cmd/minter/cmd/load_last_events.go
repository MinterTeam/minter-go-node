package cmd

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/api/v2/service"
	"github.com/MinterTeam/minter-go-node/cmd/utils"
	"github.com/MinterTeam/minter-go-node/coreV2/appdb"
	eventsdb "github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/minter"
	"github.com/spf13/cobra"
)

var (
	LoadLastEventsCommand = &cobra.Command{
		Use:   "load_last_events",
		Short: "Minter debug last events",
		RunE:  getLastEvents,
	}
)

func getLastEvents(cmd *cobra.Command, args []string) error {
	homeDir, err := cmd.Flags().GetString("home-dir")
	if err != nil {
		return err
	}
	storages := utils.NewStorage(homeDir, "")

	_, err = storages.InitEventLevelDB("data/events", minter.GetDbOpts(1024))
	if err != nil {
		return err
	}

	eventsDB := eventsdb.NewEventsStore(storages.EventDB())

	db := appdb.NewAppDB(storages.GetMinterHome(), cfg)
	height := uint32(db.GetLastHeight())
	events := eventsDB.LoadEvents(height)

	fmt.Println("height", height)
	for _, event := range events {
		fmt.Printf("%s %v\n", event.Type(), service.DecodeEvent(event))
	}
	//ldb, err := storages.InitStateLevelDB("data/state", nil)
	//if err != nil {
	//	log.Panicf("Cannot load db: %s", err)
	//}

	//currentState, err := state.NewCheckStateAtHeightV3(height, ldb)
	//if err != nil {
	//	log.Panicf("Cannot new state at given height: %s, last available height %d", err, db.GetLastHeight())
	//}

	return nil
}

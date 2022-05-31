package tests

import (
	"fmt"
	"github.com/MinterTeam/minter-go-node/coreV2/code"
	"github.com/MinterTeam/minter-go-node/coreV2/events"
	"github.com/MinterTeam/minter-go-node/coreV2/minter"
	"github.com/MinterTeam/minter-go-node/coreV2/transaction"
	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"testing"
	"time"
)

func TestUpdate(t *testing.T) {
	for _, valCount := range []int{1, 4, 20, 50, 70} {
		t.Run(fmt.Sprintf("%dvals", valCount), func(t *testing.T) {
			testUpdate(t, valCount, minter.V310, minter.V320, minter.V330)
		})
	}
}

func testUpdate(t *testing.T, valCount int, versions ...string) {
	var voters []*User
	var addresses []types.Address
	for i := 0; i < valCount; i++ {
		voter := CreateAddress()
		voters = append(voters, voter)
		addresses = append(addresses, voter.address)
	}

	helper := NewHelper(DefaultAppState(addresses...))

	for _, version := range versions {
		t.Run(version, func(t *testing.T) {
			var txs []transaction.Transaction
			height := helper.app.Height() + 5
			for i, voter := range voters {
				txs = append(txs, helper.CreateTx(voter.privateKey, transaction.VoteUpdateDataV230{
					Version: version,
					PubKey:  types.Pubkey{byte(i)},
					Height:  height,
				}, types.USDTID))
			}

			_, results := helper.NextBlock(txs...)

			for _, resp := range results {
				if resp.Code != code.OK {
					t.Errorf("Response code is not OK: %d, %s", resp.Code, resp.Log)
				}
			}

			updatedHeight := height
			for h := uint64(0); h <= updatedHeight+1; h, _ = helper.NextBlock() {
			}

			var updated bool
			for _, event := range helper.app.GetEventsDB().LoadEvents(uint32(updatedHeight)) {
				if event.Type() == events.TypeUpdateNetworkEvent {
					updateNetworkEvent, ok := event.(*events.UpdateNetworkEvent)
					if !ok {
						t.Error("incorrect event type")
						continue
					}
					if updateNetworkEvent.Version != version {
						t.Error("incorrect version", updateNetworkEvent.Version)
						continue
					}
					updated = true
				}
			}

			if !updated {
				t.Error("network is not updated")
			}

			testBlock50(t, helper)
		})
	}
}

func testBlock50(t *testing.T, helper *Helper) {
	initial := helper.app.Height()

	c := make(chan struct{})
	go func() {
		for h := uint64(0); h <= initial+50; h, _ = helper.NextBlock() {
		}
		c <- struct{}{}
	}()

	select {
	case <-c:
		return
	case <-time.After(10 * time.Second):
		t.Fatal("deadline")
		return
	}
}

package api
import (
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"time"
	)

func DurationTime(height1 int64, height2 int64)(time.Duration) {
		block, _ := client.Block(&height1)
		firstheighttime:=block.Block.Time

		block, _ = client.Block(&height2)
		secondheighttime:=block.Block.Time

	duration:=firstheighttime.Sub(secondheighttime)

return duration
} 

func HeightByTime(height int64, query string) (int64, error) {
	var deflection time.Duration
	var targettime time.Time
	var targetheight int64

	if height > 0 {
		if height > int64(blockchain.Height()) {
			return 0, rpctypes.RPCError{Code: 404, Message: "Inputed block higher actual block"}
		}
	}else{
		height = int64(blockchain.Height())
	}

	duration:=DurationTime(height, height - int64(1))

	block, _ := client.Block(&height)

	if query=="day"{
		deflection=86400000000000/duration
		targettime =block.Block.Time.AddDate(0, 0, -1) 
	} else if query=="week"{
		deflection=604800000000000/duration
		targettime =block.Block.Time.AddDate(0, 0, -7) 
	} else if query==""{
		return height, nil
	}else{
		//2020-03-28T10:21:55Z
		targettime,_ =time.Parse(time.RFC3339Nano, query)
		deflection= block.Block.Time.Sub(targettime) / duration
	}

	calcheigth:=height 

	for {
		if int64(deflection) > calcheigth{
			return 0, rpctypes.RPCError{Code: 404, Message: "Incorrect Height time"}
		}

		calcheigth = calcheigth - int64(deflection)
		block, _ = client.Block(&calcheigth) 

		duration = DurationTime(calcheigth, calcheigth - int64(1))
		deflection = block.Block.Time.Sub(targettime) / duration 

		if deflection == 0 {
			if block.Block.Time.Sub(targettime) > 0 {
				targetheight=calcheigth - int64(1)
			} else {
				targetheight=calcheigth
			}
			break
		}
	} 

	return targetheight, nil 
}
 
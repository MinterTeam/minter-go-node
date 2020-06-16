package api

import (
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"time"
)

func DurationTime(height int64, count int64) time.Duration {
	if count < 2 {
		count = 120
	}
	block1, _ := client.Block(&height)
	time1 := block1.Block.Time

	height2 := height - count
	block2, _ := client.Block(&height2)
	duration := time1.Sub(block2.Block.Time)

	return time.Duration(duration.Nanoseconds() / count)
}

func HeightByTime(query string, height int64) (int64, error) {

	h := int64(blockchain.Height())
	if height > h {
		return 0, rpctypes.RPCError{Code: 404, Message: "Inputed block higher actual block"}
	}
	if height == 0 {
		height = h
	}

	duration := DurationTime(height, 120)

	block, err := client.Block(&height)
	if err != nil {
		return 0, err
	}

	var sub time.Duration
	switch query {
	case "day":
		sub = time.Hour * 24
	case "week":
		sub = time.Hour * 24 * 7
	case "":
		return height, nil
	default:
		target, err := time.Parse(time.RFC3339, query)
		if err != nil {
			return 0, rpctypes.RPCError{Code: 404, Message: "Incorrect query time", Data: err.Error()}
		}
		sub = block.Block.Time.Sub(target)
	}

	difference := -float64(sub / duration)
	targetTime := block.Block.Time.Add(-sub)

	for {
		height += int64(difference)
		block2, err := client.Block(&height)
		if err != nil {
			return 0, err
		}
		result := targetTime.Sub(block2.Block.Time)
		difference = float64(result) / float64(DurationTime(height, 2))
		if int(difference) == 0 {
			if difference > 0 {
				return height, nil
			}
			return height - 1, nil
		}
	}
}

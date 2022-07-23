package service

import (
	"log"
	"math/big"
	"net/http"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/gin-gonic/gin"
)

func (s *Service) customExample(c *gin.Context) {
	coin0S := c.Param("coin0")
	coin1S := c.Param("coin1")
	priceS := c.Param("price")
	coin0I, _ := strconv.Atoi(coin0S)
	coin1I, _ := strconv.Atoi(coin1S)
	priceF, _ := strconv.ParseFloat(priceS, 10)
	log.Println(coin0I, coin1I, priceF)
	amount0, amount1 := s.blockchain.CurrentState().Swap().GetSwapper(types.CoinID(coin1I), types.CoinID(coin0I)).CalculateAddAmountsForPrice(big.NewFloat(1 / priceF))
	c.JSON(200, gin.H{
		"amount0": amount0,
		"amount1": amount1,
	})
}

// CustomHandlers return custom http methods
func (s *Service) CustomHandlers() http.Handler {
	r := gin.Default()
	r.GET("/swap_pool/:coin0/:coin1/:price", s.customExample)
	return r
}

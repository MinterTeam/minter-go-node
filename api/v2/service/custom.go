package service

import (
	"math/big"
	"net/http"
	"strconv"

	"github.com/MinterTeam/minter-go-node/coreV2/types"
	"github.com/gin-gonic/gin"
)

func (s *Service) changeAmountsForPrice(c *gin.Context) {
	coin0S := c.Param("coin0")
	coin1S := c.Param("coin1")
	priceS := c.Param("price")
	coin0I, err := strconv.Atoi(coin0S)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": err.Error(),
			},
		})
		return
	}
	coin1I, err := strconv.Atoi(coin1S)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": err.Error(),
			},
		})
		return
	}
	priceF, err := strconv.ParseFloat(priceS, 10)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": map[string]string{
				"message": err.Error(),
			},
		})
		return
	}
	amount0, amount1 := s.blockchain.CurrentState().Swap().GetSwapper(types.CoinID(coin0I), types.CoinID(coin1I)).CalculateAddAmountsForPrice(big.NewFloat(1 / priceF))
	c.JSON(200, gin.H{
		"amount0In":  amount0,
		"amount1Out": amount1,
		// todo: reverse price if amounts nil
	})
}

// CustomHandlers return custom http methods
func (s *Service) CustomHandlers() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.GET("/change_amounts_for_price/:coin0/:coin1/:price", s.changeAmountsForPrice)
	return r
}

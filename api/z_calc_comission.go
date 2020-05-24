package api

import (
	"github.com/MinterTeam/minter-go-node/formula"
	"github.com/MinterTeam/minter-go-node/rpc/lib/types"
	"github.com/MinterTeam/minter-go-node/core/types"
	"github.com/MinterTeam/minter-go-node/core/transaction"
	"github.com/MinterTeam/minter-go-node/core/commissions"
	"math/big"
)

type UseMaxResponse struct {
	GasCoin 		string `json:"gascoin"`
	StartValue 		string `json:"startvalue"`
	TXComissionValue 	string `json:"txvalue"`
	EndValue 		string `json:"endvalue"`
}


func CalcTxCommission(gascoin string, height int, txtype string, payload string, mtxs int) (string, error) {
	commissionInBaseCoin:=big.NewInt(0)
	if txtype == "SendTx" {commissionInBaseCoin = big.NewInt(commissions.SendTx)}
	if txtype == "ConvertTx" {commissionInBaseCoin = big.NewInt(commissions.ConvertTx)}
	if txtype == "DeclareCandidacyTx" {commissionInBaseCoin = big.NewInt(commissions.DeclareCandidacyTx)}
	if txtype == "DelegateTx" {commissionInBaseCoin = big.NewInt(commissions.DelegateTx)}
	if txtype == "UnbondTx" {commissionInBaseCoin = big.NewInt(commissions.UnbondTx)}
	if txtype == "ToggleCandidateStatus" {commissionInBaseCoin = big.NewInt(commissions.ToggleCandidateStatus)}
	if txtype == "EditCandidate" {commissionInBaseCoin = big.NewInt(commissions.EditCandidate)}
	if txtype == "RedeemCheckTx" {commissionInBaseCoin = big.NewInt(commissions.RedeemCheckTx)}
	if txtype == "CreateMultisig" {commissionInBaseCoin = big.NewInt(commissions.CreateMultisig)}
	if txtype == "MultiSend" {
	if mtxs == 0 {
		return "", rpctypes.RPCError{Code: 400, Message: "Set number of txs for multisend (mtxs)"}
	}
	commissionInBaseCoin = big.NewInt(commissions.MultisendDelta*(int64(mtxs)+1))
	}
	
	if commissionInBaseCoin.Cmp(big.NewInt(0))== 0{
		return "", rpctypes.RPCError{Code: 401, Message: "Set correct txtype for tx"}
	}

	commissionInBaseCoin.Mul(commissionInBaseCoin, transaction.CommissionMultiplier)
	commissionfreepayload := big.NewInt(0).Set(commissionInBaseCoin)
	
	payloadbyte := 1000
	if len([]byte(payload)) < 1000 {
		payloadbyte = len([]byte(payload))
	}

	payloadcomission:= big.NewInt(commissions.PayloadByte * int64(payloadbyte))
	payloadcomission.Mul(payloadcomission, transaction.CommissionMultiplier)
	comissionpayload := big.NewInt(0).Set(payloadcomission)

	totalCommissionInBaseCoin := new(big.Int).Add(commissionfreepayload,comissionpayload)

	cState, err := GetStateForHeight(height)
	if err != nil {
		return "", err
	}

	cState.RLock() 
	defer cState.RUnlock()
	var commission *big.Int
	
	if gascoin != "BIP" {

		coin := cState.Coins.GetCoin(types.StrToCoinSymbol(gascoin))

		if coin == nil {
			return "", rpctypes.RPCError{Code: 404, Message: "Coin not found"}
		}

		if totalCommissionInBaseCoin.Cmp(coin.Reserve()) == 1{
			return "", rpctypes.RPCError{Code: 400, Message: "Not enough coin reserve for pay comission"}
		}

		commission= formula.CalculateSaleAmount(coin.Volume(), coin.Reserve(), coin.Crr(), totalCommissionInBaseCoin )
	}else{
		commission= totalCommissionInBaseCoin 
	}
	return commission.String(), nil
}
func CalcFreeCoinForTx(gascoin string, gascoinamount big.Int, height int, txtype string, payload string, mtxs int) (UseMaxResponse, error) {

	comission,err:=CalcTxCommission(gascoin,height,txtype,payload,mtxs)
	
	if err != nil {
		return UseMaxResponse{}, err
	}

	commissionBig := new(big.Int) 
    	commissionBig.SetString(comission, 10) 
	
	if gascoinamount.Cmp(commissionBig) == -1{
		return UseMaxResponse{}, rpctypes.RPCError{Code: 400, Message: "Not enough coin bipvalue for pay comission"}
	}

	return UseMaxResponse {
		GasCoin: 		gascoin,
		StartValue: 		gascoinamount.String(),
		TXComissionValue: 	comission,
		EndValue: 		big.NewInt(0).Sub(&gascoinamount,commissionBig).String(),
	}, nil  
}
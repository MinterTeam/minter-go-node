package service

import (
	"context"
	pb "github.com/MinterTeam/node-grpc-gateway/api_pb"
	"github.com/golang/protobuf/ptypes/empty"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"
)

// TestBlock returns test block.
func (s *Service) TestBlock(context.Context, *empty.Empty) (*pb.BlockResponse, error) {
	anySendData, err := anypb.New(&pb.SendData{
		Coin: &pb.Coin{
			Id:     1,
			Symbol: "CUSTOM",
		},
		To:    "Mxa83d8ebbe688b853775a698683b77afa305a661e",
		Value: "1000000000000000000",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anySellData, err := anypb.New(&pb.SellCoinData{
		CoinToBuy: &pb.Coin{
			Id:     1,
			Symbol: "CUSTOM",
		},
		CoinToSell: &pb.Coin{
			Id:     2,
			Symbol: "CUSTOM2",
		},
		MinimumValueToBuy: "1363431564908222940563",
		ValueToSell:       "400000000000000000000",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anySellAllData, err := anypb.New(&pb.SellAllCoinData{
		CoinToBuy: &pb.Coin{
			Id:     1,
			Symbol: "CUSTOM",
		},
		CoinToSell: &pb.Coin{
			Id:     2,
			Symbol: "CUSTOM2",
		},
		MinimumValueToBuy: "1363431564908222940563",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyBuyData, err := anypb.New(&pb.BuyCoinData{
		CoinToBuy: &pb.Coin{
			Id:     1,
			Symbol: "CUSTOM",
		},
		CoinToSell: &pb.Coin{
			Id:     2,
			Symbol: "CUSTOM2",
		},
		MaximumValueToSell: "1363431564908222940563",
		ValueToBuy:         "400000000000000000000",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyCreateCoinData, err := anypb.New(&pb.CreateCoinData{
		Name:                 "Custom Coin 2",
		Symbol:               "CUSTOM2",
		InitialAmount:        "1234567000000000000000000",
		InitialReserve:       "123456000000000000000000",
		ConstantReserveRatio: 45,
		MaxSupply:            "12345679000000000000000000",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyDeclareCandidacyData, err := anypb.New(&pb.DeclareCandidacyData{
		Address:    "Mxa83d8ebbe688b853775a698683b77afa305a661e",
		PubKey:     "Mp629b5528f09d1c74a83d18414f2e4263e14850c47a3fac3f855f200111111111",
		Commission: 10,
		Coin: &pb.Coin{
			Id:     1,
			Symbol: "CUSTOM",
		},
		Stake: "2000000000000000000000",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyDelegateData, err := anypb.New(&pb.DelegateData{
		Coin: &pb.Coin{
			Id:     1,
			Symbol: "CUSTOM",
		},
		PubKey: "Mp629b5528f09d1c74a83d18414f2e4263e14850c47a3fac3f855f200111111111",
		Value:  "200000000000000000000",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyUnbondData, err := anypb.New(&pb.UnbondData{
		Coin: &pb.Coin{
			Id:     1,
			Symbol: "CUSTOM",
		},
		PubKey: "Mp95b76c6893dc28a34f005b9708bac59eae238232ef86798d672387bbb849bd22",
		Value:  "8975000000000000000000",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyRedeemCheckData, err := anypb.New(&pb.RedeemCheckData{
		RawCheck: "+JqDNTY3AYMBH7uAiA3gtrOnZAAAgLhB+dC7WoUyqC62WxY7iVx0PL/mhBILaw2p/cADqXquPkYVyzBGo5qfEqkFPXtOQgBd/TBHub3u0YnEJRpqxovgAgAboBzjalk0Q+HNhOqTjzRUFeQuZni4ZJEhJETxAdTLYeK3oHD01JHIGkUUzVywMDZUprmmrNYfnJvihAQpclhdwpit",
		Proof:    "OwpFFzYUFOXPkhu+6TEpX5XRZ6ShoCyqUbPO/CU0zotY50Y1lHJX+zbOiQMeI/5mQ21cqT9BBHsmtgFWPSa1wgE=",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anySetCandidateOnData, err := anypb.New(&pb.SetCandidateOnData{
		PubKey: "Mp95b76c6893dc28a34f005b9708bac59eae238232ef86798d672387bbb849bd22",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anySetCandidateOffData, err := anypb.New(&pb.SetCandidateOffData{
		PubKey: "Mpb451f898f2d5e054b9edc4b06c2cbcf1c318348593a05cae32565ec665758207",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyCreateMultisigData, err := anypb.New(&pb.CreateMultisigData{
		Threshold: 1,
		Weights:   []uint64{1, 1},
		Addresses: []string{"Mxe8eff43f860f82ba60c7ba58c47d898462173eee", "Mxb4b0fe832afc10c700ad1a73f3f109b574233ee1"},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyMultiSendData, err := anypb.New(&pb.MultiSendData{List: []*pb.SendData{
		{
			Coin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			To:    "Mxa83d8ebbe688b853775a698683b77afa305a661e",
			Value: "12000000000000000000",
		},
		{
			Coin: &pb.Coin{
				Id:     2,
				Symbol: "CUSTOM2",
			},
			To:    "Mxa83d8ebbe688b853775a698683b77afa305a661e",
			Value: "2000000000000000000",
		},
	}})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyEditCandidateData, err := anypb.New(&pb.EditCandidateData{
		PubKey:         "Mp5e3e1da62c7eabd9d8d168a36a92727fc1970a54ec61eadd285d4199c41191d7",
		RewardAddress:  "Mx72f82017a10d095ca697db6f1fa86229a000feed",
		OwnerAddress:   "Mx4c40df842347c3edbc4b6e6183116825e4000cab",
		ControlAddress: "Mxbddf9768aca06e980570670d912f6f6481ce0c95",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anySetHeltData, err := anypb.New(&pb.SetHaltBlockData{
		PubKey: "Mp5e3e1da62c7eabd9d8d168a36a92727fc1970a54ec61eadd285d4199c41191d7",
		Height: 1112,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyRecreateCoinData, err := anypb.New(&pb.RecreateCoinData{
		Name:                 "CHAIN",
		Symbol:               "CHAIN",
		InitialAmount:        "10000000000000000000000",
		InitialReserve:       "10000000000000000000000",
		ConstantReserveRatio: 100,
		MaxSupply:            "10000000000000000000000",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyEditCoinOwnerData, err := anypb.New(&pb.EditCoinOwnerData{
		Symbol:   "CUSTOM",
		NewOwner: "Mx72f82017a10d095ca697db6f1fa86229a000feed",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyEditMultisigData, err := anypb.New(&pb.EditMultisigData{
		Threshold: 2,
		Weights:   []uint64{1, 1},
		Addresses: []string{"Mxe8eff43f860f82ba60c7ba58c47d898462173eee", "Mxb4b0fe832afc10c700ad1a73f3f109b574233ee1"},
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyPriceVoteData, err := anypb.New(&pb.PriceVoteData{
		Price: "1000",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	anyEditCandidatePublicKeyData, err := anypb.New(&pb.EditCandidatePublicKeyData{
		PubKey:    "Mp5e3e1da62c7eabd9d8d168a36a92727fc1970a54ec61eadd285d4199c41191d7",
		NewPubKey: "Mpb451f898f2d5e054b9edc4b06c2cbcf1c318348593a05cae32565ec665758207",
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	transactions := []*pb.TransactionResponse{
		{
			Hash:        "Mt88d02883c07a1dfc5ff5b2d27eacfd0d82706ba113f2e77a42a1a3d3c369c249",
			RawTx:       "f8700102018001a0df0194a83d8ebbe688b853775a698683b77afa305a661e880de0b6b3a7640000808001b845f8431ca0acde2fa28bf063bffb14f667a2219b641205bd67a5bc6b664cbd44a62504c897a03649ae0ae4881426a95ccc80866c259032f22811777777c7327e9b99f766ad00",
			Height:      123,
			Index:       1,
			From:        "Mx0c5d5f646556d663e1eaf87150d987b9f2b858b6",
			Nonce:       1,
			GasPrice:    1,
			Type:        1,
			Data:        anySendData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         10,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.coin_id": "1",
				"tx.from":    "0c5d5f646556d663e1eaf87150d987b9f2b858b6",
				"tx.to":      "a83d8ebbe688b853775a698683b77afa305a661e",
				"tx.type":    "01",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt2007d1e926f1a272228acd547f63d5317060f471da98d63eead28ab476a7cbc2",
			RawTx:       "f8608204c0010180028ecd808901a9e1239779e80e2e1580808001b845f8431ba059477ac6f7bf274104ab155710dbe4069ed308367079ec78577a4bdbf6fb9ae5a02e710ff2bd48f52f880d611141ec03f7083e23e874cf6316dd6ff6c1b4eb703b",
			From:        "Mxa167c767cb47531018c475874e68dee0a7cd6ecf",
			Nonce:       1216,
			GasPrice:    1,
			Type:        2,
			Data:        anySellData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         100,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.coin_to_buy":  "1",
				"tx.coin_to_sell": "2",
				"tx.from":         "a167c767cb47531018c475874e68dee0a7cd6ecf",
				"tx.return":       "12899562838043362652",
				"tx.type":         "02",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mta03e4cd8576dbb941ea65c0888237e627a3081ac2211e40e74293a1894ccc7b7",
			RawTx:       "f855730101800385c480819c80808001b845f8431ca008ad0b44f88fe05fd8e50a2e3ab5d02be6e073c54b0d0badd683e8cd6e052eb8a03ce32a7214300afc56e3982d380162028d91d3918b4273a521f3cbf139d44bcc",
			From:        "Mx9dc6fd7c76cf6d86c708632142bed27efed9f421",
			Nonce:       115,
			GasPrice:    1,
			Type:        3,
			Data:        anySellAllData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         100,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.coin_to_buy":  "1",
				"tx.coin_to_sell": "2",
				"tx.from":         "9dc6fd7c76cf6d86c708632142bed27efed9f421",
				"tx.return":       "497502973808670417789",
				"tx.sell_amount":  "27128313394438470654",
				"tx.type":         "03",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt5414921f355af83aa4610312b6f31bcff3c3bbc274b5161e16305114c349690b",
			RawTx:       "f8698202830101800497d61f8915af1d78b58c400000808949e969f88395930593808001b845f8431ba05d96f7c8f114e220601cd159426e8e4b8937e16316a17f8f2a97593e521263c7a0114816c293966a041ca6d02fac601a243ec4602b7a9d1928c8660a934abb85dc",
			From:        "Mxde774d73bcd38b7c0108ea7406658c40df5ba369",
			Nonce:       643,
			GasPrice:    1,
			Type:        4,
			Data:        anyBuyData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         100,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.coin_to_buy":  "31",
				"tx.coin_to_sell": "0",
				"tx.from":         "de774d73bcd38b7c0108ea7406658c40df5ba369",
				"tx.return":       "1239483240825657218694",
				"tx.type":         "04",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt7f407505c6fad554b4517ede32058009e5062d9d8a7da68b99cdfe0cc8878980",
			RawTx:       "f88e0102018005b83df83b8b437573746f6d20436f696e8a435553544f4d000000008b01056e02dc4bb2ddbc00008a1a24902bee14210000002d8b0a364c9981614d8bdc0000808001b845f8431ca0140c27a6340680a028baa0761bdb43bf9bd575c541b8107ee378e584fdf92780a04d2ef3b55303442a88564c59e1ee0255a24e78b7e57b62e4121705bbefdab5a7",
			From:        "Mxa83d8ebbe688b853775a698683b77afa305a661e",
			Nonce:       1,
			GasPrice:    1,
			Type:        5,
			Data:        anyCreateCoinData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         100,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.coin_id":     "2",
				"tx.coin_symbol": "CUSTOM2",
				"tx.from":        "a83d8ebbe688b853775a698683b77afa305a661e",
				"tx.type":        "05",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt7f407505c6fad554b4517ede32058009e5062d9d8a7da68b99cdfe0cc8878980",
			RawTx:       "f88e0102018005b83df83b8b437573746f6d20436f696e8a435553544f4d000000008b01056e02dc4bb2ddbc00008a1a24902bee14210000002d8b0a364c9981614d8bdc0000808001b845f8431ca0140c27a6340680a028baa0761bdb43bf9bd575c541b8107ee378e584fdf92780a04d2ef3b55303442a88564c59e1ee0255a24e78b7e57b62e4121705bbefdab5a7",
			From:        "Mxa83d8ebbe688b853775a698683b77afa305a661e",
			Nonce:       1,
			GasPrice:    1,
			Type:        6,
			Data:        anyDeclareCandidacyData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         100,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "a83d8ebbe688b853775a698683b77afa305a661e",
				"tx.type": "06",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt5ea36034fa71c471c8bc05f561ef94e33cffc79f8b5d331cc4ea14a0fca259ad",
			RawTx:       "f88082175001018007aeeda0629b5528f09d1c74a83d18414f2e4263e14850c47a3fac3f855f200111111111808a01e6892544697adc0000808001b845f8431ba0ba3c0b27367d6635624d8c5c20d215f17199cc2e9571e0f957a706f620bb7d49a0558097bd295ed2f5f97adfc8ee1cc7d3791c19171388611263fa9f363c776323",
			From:        "Mx965a21571aa6fac1de7b347e8e6d94e4d83c31a7",
			Nonce:       5968,
			GasPrice:    1,
			Type:        7,
			Data:        anyDelegateData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         200,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "965a21571aa6fac1de7b347e8e6d94e4d83c31a7",
				"tx.type": "07",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mtf7ef9f03ae8aac7d0e33bd7eea24ec0e7b06f5be08eef9eb6f6681c13448474e",
			RawTx:       "f87d6f01018008adeca095b76c6893dc28a34f005b9708bac59eae238232ef86798d672387bbb849bd2280890ad78ebc5ac6200000808001b845f8431ca0aaf1c3bb88ca61a24858c0cb7b6ecf44eb9c9f6c68771b7bde1bee6d490a5297a03b5467762ff56010aff78ee51ac7ee079b6820ef7a2e6e4d79b8a76d0085ca21",
			From:        "Mxceee14ba87144c3b327563ba8c6154b2f8c622c1",
			Nonce:       111,
			GasPrice:    1,
			Type:        8,
			Data:        anyUnbondData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         200,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "ceee14ba87144c3b327563ba8c6154b2f8c622c1",
				"tx.type": "08",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt739c556502c198f7434be94a16949790c41711d5a6540d8936a749ff131dfa1a",
			RawTx:       "f9013582197c01018009b8e3f8e1b89cf89a833536370183011fbb80880de0b6b3a764000080b841f9d0bb5a8532a82eb65b163b895c743cbfe684120b6b0da9fdc003a97aae3e4615cb3046a39a9f12a9053d7b4e42005dfd3047b9bdeed189c4251a6ac68be002001ba01ce36a593443e1cd84ea938f345415e42e6678b86491212444f101d4cb61e2b7a070f4d491c81a4514cd5cb0303654a6b9a6acd61f9c9be284042972585dc298adb8413b0a4517361414e5cf921bbee931295f95d167a4a1a02caa51b3cefc2534ce8b58e74635947257fb36ce89031e23fe66436d5ca93f41047b26b601563d26b5c201808001b844f8421ca096ed752cd10e8e7bd0656cb316ac43fc6ab3944da4a46da382863acd8f3132e09f805734e10c482ca6be7dd8508e0da768c24b7a05123d41b9312cb6bb5f55f9",
			From:        "Mx777061cc8595f5aa3785956b318cfec4c8fc4777",
			Nonce:       6524,
			GasPrice:    1,
			Type:        9,
			Data:        anyRedeemCheckData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         30,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.coin_id": "0",
				"tx.from":    "66666629de464a22563478b39bd1b184f4134e5a",
				"tx.to":      "777061cc8595f5aa3785956b318cfec4c8fc4777",
				"tx.type":    "09",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt105e72a66339cfa98ef3b40bc94d969b1ee576755aec2095e467f83ae9e8386a",
			RawTx:       "f8748201370101800aa2e1a0b451f898f2d5e054b9edc4b06c2cbcf1c318348593a05cae32565ec665758207808001b845f8431ca0ce880585421e82eae2f115de9e1135a6effc0816b88e54d1e6db6788908eea19a00b01417496184859831bc40731eec8174b9cb5f7575afa19dea35e1ef80160ac",
			From:        "Mx4ec1587f74bee9b4251cd5158a1b09849cd33725",
			Nonce:       311,
			GasPrice:    1,
			Type:        10,
			Data:        anySetCandidateOnData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         100,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "4ec1587f74bee9b4251cd5158a1b09849cd33725",
				"tx.type": "0a",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt36fcf38d74866b0b88ce4d9222808b4ea4b4ae85277ddee6d459dda6a152e887",
			RawTx:       "f8748201360101210ba2e1a0b451f898f2d5e054b9edc4b06c2cbcf1c318348593a05cae32565ec665758207808001b845f8431ca010173191a7e118b7a18af13e279521a058e310b1b73fd932c0414fd053fa1eaca02aa1091ae7edcfb296b41b7a29e0467bd6035eb3d232dcc1d88b8da9920be947",
			From:        "Mx4ec1587f74bee9b4251cd5158a1b09849cd33725",
			Nonce:       310,
			GasPrice:    1,
			Type:        11,
			Data:        anySetCandidateOffData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         100,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "4ec1587f74bee9b4251cd5158a1b09849cd33725",
				"tx.type": "0b",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt4b513e64ccc92ac33040e1b8e34b83043fd2164fd2777877ec1dc448c299f074",
			RawTx:       "f88282039b0101800cb0ef01c20101ea94e8eff43f860f82ba60c7ba58c47d898462173eee94b4b0fe832afc10c700ad1a73f3f109b574233ee1808001b845f8431ca03f4b19c2f9b9d793af7b2fb55de0d971ffde19beaaee6b2ed7543dcf1a189b14a00ab6c16d2c0973e3c97e9b202120590457f1bdcde4612d3a6ff039b297a6a9b5",
			From:        "Mxe8eff43f860f82ba60c7ba58c47d898462173eee",
			Nonce:       923,
			GasPrice:    1,
			Type:        12,
			Data:        anyCreateMultisigData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         100,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.created_multisig": "590fd3671b226923c64e26e4e51094c027a71b4a",
				"tx.from":             "e8eff43f860f82ba60c7ba58c47d898462173eee",
				"tx.type":             "0c",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mta588b1593b8d4e9c0eba47897d8a3c05513ed1ac018d121dca9f75c380ff7c59",
			RawTx:       "f895010201800db844f842f840df0194a83d8ebbe688b853775a698683b77afa305a661e88a688906bd8b00000df8094a83d8ebbe688b853775a698683b77afa305a661e881bc16d674ec80000808001b845f8431ca02d241cad1b60950109ddbe9f0b24fa7c26a39d2c2b37e355598128511fc352f5a03e5ecae4acd59ee191c39cc366e726387c0d9ebfc5cf35bd347893afe551e48d",
			From:        "Mx0fa4821eba5dcc20e71c02208cbc7b255878ab6b",
			Nonce:       1,
			GasPrice:    1,
			Type:        13,
			Data:        anyMultiSendData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         15,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "0fa4821eba5dcc20e71c02208cbc7b255878ab6b",
				"tx.to":   "a83d8ebbe688b853775a698683b77afa305a661e,a83d8ebbe688b853775a698683b77afa305a661e",
				"tx.type": "0d",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mtf8022f85fd38e975b08cdde09de598e2ca89f694c2c98720afca476554e90b67",
			RawTx:       "f8b3320101800eb862f860a05e3e1da62c7eabd9d8d168a36a92727fc1970a54ec61eadd285d4199c41191d79472f82017a10d095ca697db6f1fa86229a000feed944c40df842347c3edbc4b6e6183116825e4000cab94bddf9768aca06e980570670d912f6f6481ce0c95808001b845f8431ca0d1a14af3bb91e2f79e837a008ea4daf72f81ce699628a7426ff513d5b1babdf6a0610e3a0f6782e20beaccb3cbffb79578a2c378f5291394f908b7a67fc03ca0ce",
			From:        "Mx4c40df842347c3edbc4b6e6183116825e4000cab",
			Nonce:       50,
			GasPrice:    1,
			Type:        14,
			Data:        anyEditCandidateData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         10000,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "4c40df842347c3edbc4b6e6183116825e4000cab",
				"tx.type": "0e",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mtf8022f85fd38e975b08cdde09de598e2ca89f694c2c98720afca476554e90b67",
			RawTx:       "f8b3320101800eb862f860a05e3e1da62c7eabd9d8d168a36a92727fc1970a54ec61eadd285d4199c41191d79472f82017a10d095ca697db6f1fa86229a000feed944c40df842347c3edbc4b6e6183116825e4000cab94bddf9768aca06e980570670d912f6f6481ce0c95808001b845f8431ca0d1a14af3bb91e2f79e837a008ea4daf72f81ce699628a7426ff513d5b1babdf6a0610e3a0f6782e20beaccb3cbffb79578a2c378f5291394f908b7a67fc03ca0ce",
			From:        "Mx4c40df842347c3edbc4b6e6183116825e4000cab",
			Nonce:       50,
			GasPrice:    1,
			Type:        15,
			Data:        anySetHeltData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         10000,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "4c40df842347c3edbc4b6e6183116825e4000cab",
				"tx.type": "0f",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt2216fe1c11129fdc7f0a15692198d4287cb083a10288503f80bee87f8006c293",
			RawTx:       "f8841001018010b4f385434841494e8a434841494e00000000008a021e19e0c9bab24000008a021e19e0c9bab2400000648a021e19e0c9bab2400000808001b845f8431ca0cd85169a87359efac68d0c9d3bc895b4d577956bd58223ff9fdf293550b218c4a05ac5c18c030244f568ea7e3b2893681c05dfb0c2631decd598c5f1942f4010cc",
			From:        "Mx1434ec1028936cd49ce82c88278e43b8e888100a",
			Nonce:       50,
			GasPrice:    1,
			Type:        16,
			Data:        anyRecreateCoinData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         10000000,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.coin_id":         "1675",
				"tx.coin_symbol":     "CHAIN",
				"tx.old_coin_symbol": "16",
				"tx.old_coin_id":     "CHAIN-3",
				"tx.from":            "1434ec1028936cd49ce82c88278e43b8e888100a",
				"tx.type":            "10",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt2216fe1c11129fdc7f0a15692198d4287cb083a10288503f80bee87f8006c293",
			RawTx:       "f8841001018010b4f385434841494e8a434841494e00000000008a021e19e0c9bab24000008a021e19e0c9bab2400000648a021e19e0c9bab2400000808001b845f8431ca0cd85169a87359efac68d0c9d3bc895b4d577956bd58223ff9fdf293550b218c4a05ac5c18c030244f568ea7e3b2893681c05dfb0c2631decd598c5f1942f4010cc",
			From:        "Mx1434ec1028936cd49ce82c88278e43b8e888100a",
			Nonce:       50,
			GasPrice:    1,
			Type:        17,
			Data:        anyEditCoinOwnerData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         10000000,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from":        "1434ec1028936cd49ce82c88278e43b8e888100a",
				"tx.type":        "11",
				"tx.coin_symbol": "CUSTOM",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt2216fe1c11129fdc7f0a15692198d4287cb083a10288503f80bee87f8006c293",
			RawTx:       "f8841001018010b4f385434841494e8a434841494e00000000008a021e19e0c9bab24000008a021e19e0c9bab2400000648a021e19e0c9bab2400000808001b845f8431ca0cd85169a87359efac68d0c9d3bc895b4d577956bd58223ff9fdf293550b218c4a05ac5c18c030244f568ea7e3b2893681c05dfb0c2631decd598c5f1942f4010cc",
			From:        "Mx1434ec1028936cd49ce82c88278e43b8e888100a",
			Nonce:       50,
			GasPrice:    1,
			Type:        18,
			Data:        anyEditMultisigData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         10000000,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "1434ec1028936cd49ce82c88278e43b8e888100a",
				"tx.type": "12",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt9c48d70f4fb35b2b3f638ea434e7263c5136c60559015d141ec2a94b8ba346a1",
			RawTx:       "f854080101801384c38203e8808001b845f8431ba01ae016202f873c19e03fc5861b552a729b769332dfa93ff61bd8df5f1557c67ea0241bd95318796acc6fbcda88fa019047e162f0d0b9e6d55071cd88156bd04fe8",
			From:        "Mx7f7f0109005d866eeeee7bfc16adfb05703dd3d8",
			Nonce:       50,
			GasPrice:    1,
			Type:        19,
			Data:        anyPriceVoteData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         10000000,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "7f7f0109005d866eeeee7bfc16adfb05703dd3d8",
				"tx.type": "13",
			},
			Code: 0,
			Log:  "",
		},
		{
			Hash:        "Mt9c48d70f4fb35b2b3f638ea434e7263c5136c60559015d141ec2a94b8ba346a1",
			RawTx:       "f854080101801384c38203e8808001b845f8431ba01ae016202f873c19e03fc5861b552a729b769332dfa93ff61bd8df5f1557c67ea0241bd95318796acc6fbcda88fa019047e162f0d0b9e6d55071cd88156bd04fe8",
			From:        "Mx7f7f0109005d866eeeee7bfc16adfb05703dd3d8",
			Nonce:       50,
			GasPrice:    1,
			Type:        20,
			Data:        anyEditCandidatePublicKeyData,
			Payload:     []byte("Message"),
			ServiceData: nil,
			Gas:         10000000,
			GasCoin: &pb.Coin{
				Id:     1,
				Symbol: "CUSTOM",
			},
			Tags: map[string]string{
				"tx.from": "7f7f0109005d866eeeee7bfc16adfb05703dd3d8",
				"tx.type": "14",
			},
			Code: 0,
			Log:  "",
		},
	}

	return &pb.BlockResponse{
		Hash:             "54dad1ad97811c05729d1174a350433c4209482b4c3cfd04bfcc773fa47a54bb",
		Height:           3424,
		Time:             "2020-10-21T20:37:22.195700943Z",
		TransactionCount: uint64(len(transactions)),
		Transactions:     transactions,
		BlockReward:      "333000000000000000000",
		Size:             916,
		Proposer:         "Mpd83e627510eea6aefa46d9914b0715dabf4a561ced78d34267b31d41d5f700b5",
		Validators: []*pb.BlockResponse_Validator{
			{
				PublicKey: "Mpd83e627510eea6aefa46d9914b0715dabf4a561ced78d34267b31d41d5f700b5",
				Signed:    true,
			},
		},
	}, nil
}

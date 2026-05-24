package indexer

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

func TestDecodeMarketCreated(t *testing.T) {
	factory := common.HexToAddress("0x1000000000000000000000000000000000000001")
	market := common.HexToAddress("0x2000000000000000000000000000000000000002")
	creator := common.HexToAddress("0x3000000000000000000000000000000000000003")
	data, err := marketCreatedArgs.Pack(big.NewInt(1234), "BTC above 100k")
	if err != nil {
		t.Fatalf("pack data: %v", err)
	}

	decoded, err := DecodeLog(types.Log{
		Address: factory,
		Topics: []common.Hash{
			topicMarketCreated,
			common.BigToHash(big.NewInt(7)),
			common.BytesToHash(common.LeftPadBytes(market.Bytes(), 32)),
			common.BytesToHash(common.LeftPadBytes(creator.Bytes(), 32)),
		},
		Data: data,
	}, factory)
	if err != nil {
		t.Fatalf("DecodeLog() error = %v", err)
	}
	if decoded.Type != EventMarketCreated {
		t.Fatalf("event type = %s", decoded.Type)
	}

	var payload MarketCreatedPayload
	if err := json.Unmarshal(decoded.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.MarketID != "7" || payload.Market != market.Hex() || payload.Creator != creator.Hex() {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.LockTime != 1234 || payload.ResolutionRule != "BTC above 100k" {
		t.Fatalf("unexpected non-indexed payload: %+v", payload)
	}
}

func TestDecodeTradeExecuted(t *testing.T) {
	data, err := abi.Arguments{{Type: mustABIType("uint256")}}.Pack(big.NewInt(5_000))
	if err != nil {
		t.Fatalf("pack data: %v", err)
	}
	user := common.HexToAddress("0x3000000000000000000000000000000000000003")

	decoded, err := DecodeLog(types.Log{
		Topics: []common.Hash{
			topicTradeExecuted,
			common.BigToHash(big.NewInt(7)),
			common.BytesToHash(common.LeftPadBytes(user.Bytes(), 32)),
			common.BigToHash(big.NewInt(1)),
		},
		Data: data,
	}, common.Address{})
	if err != nil {
		t.Fatalf("DecodeLog() error = %v", err)
	}

	var payload TradeExecutedPayload
	if err := json.Unmarshal(decoded.Payload, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.MarketID != "7" || payload.User != user.Hex() || payload.Side != "YES" || payload.Amount != "5000" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestDecodeRejectsUnknownFactory(t *testing.T) {
	data, err := marketCreatedArgs.Pack(big.NewInt(1234), "rule")
	if err != nil {
		t.Fatalf("pack data: %v", err)
	}

	_, err = DecodeLog(types.Log{
		Address: common.HexToAddress("0x9999999999999999999999999999999999999999"),
		Topics: []common.Hash{
			topicMarketCreated,
			common.BigToHash(big.NewInt(7)),
			{},
			{},
		},
		Data: data,
	}, common.HexToAddress("0x1000000000000000000000000000000000000001"))
	if err == nil {
		t.Fatal("expected unknown factory error")
	}
}

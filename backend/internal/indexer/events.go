package indexer

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	EventMarketCreated   = "MARKET_CREATED"
	EventTradeExecuted   = "TRADE_EXECUTED"
	EventMarketLocked    = "MARKET_LOCKED"
	EventOracleSubmitted = "ORACLE_SUBMITTED"
	EventMarketResolved  = "MARKET_RESOLVED"
	EventRedeemed        = "REDEEMED"
)

var (
	topicMarketCreated   = crypto.Keccak256Hash([]byte("MarketCreated(uint256,address,address,uint256,string)"))
	topicTradeExecuted   = crypto.Keccak256Hash([]byte("TradeExecuted(uint256,address,uint8,uint256)"))
	topicMarketLocked    = crypto.Keccak256Hash([]byte("MarketLocked(uint256,uint256)"))
	topicOracleSubmitted = crypto.Keccak256Hash([]byte("OracleSubmitted(uint256,uint8,uint256,uint256)"))
	topicMarketResolved  = crypto.Keccak256Hash([]byte("MarketResolved(uint256,uint8)"))
	topicRedeemed        = crypto.Keccak256Hash([]byte("Redeemed(uint256,address,uint256)"))

	uintArg           = abi.Arguments{{Type: mustABIType("uint256")}}
	marketCreatedArgs = abi.Arguments{{Type: mustABIType("uint256")}, {Type: mustABIType("string")}}
)

type DecodedEvent struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type MarketCreatedPayload struct {
	MarketID       string `json:"marketId"`
	Market         string `json:"market"`
	Creator        string `json:"creator"`
	LockTime       uint64 `json:"lockTime"`
	ResolutionRule string `json:"resolutionRule"`
}

type TradeExecutedPayload struct {
	MarketID string `json:"marketId"`
	User     string `json:"user"`
	Side     string `json:"side"`
	Amount   string `json:"amount"`
}

type MarketLockedPayload struct {
	MarketID string `json:"marketId"`
	LockTime uint64 `json:"lockTime"`
}

type OracleSubmittedPayload struct {
	MarketID string `json:"marketId"`
	Outcome  string `json:"outcome"`
	Nonce    string `json:"nonce"`
	Expiry   uint64 `json:"expiry"`
}

type MarketResolvedPayload struct {
	MarketID string `json:"marketId"`
	Outcome  string `json:"outcome"`
}

type RedeemedPayload struct {
	MarketID string `json:"marketId"`
	User     string `json:"user"`
	Amount   string `json:"amount"`
}

func EventTopics() []common.Hash {
	return []common.Hash{
		topicMarketCreated,
		topicTradeExecuted,
		topicMarketLocked,
		topicOracleSubmitted,
		topicMarketResolved,
		topicRedeemed,
	}
}

func DecodeLog(log types.Log, factory common.Address) (DecodedEvent, error) {
	if len(log.Topics) == 0 {
		return DecodedEvent{}, errors.New("log has no topics")
	}

	switch log.Topics[0] {
	case topicMarketCreated:
		if log.Address != factory {
			return DecodedEvent{}, fmt.Errorf("market created emitted by unknown factory %s", log.Address.Hex())
		}
		return decodeMarketCreated(log)
	case topicTradeExecuted:
		return decodeTradeExecuted(log)
	case topicMarketLocked:
		return decodeMarketLocked(log)
	case topicOracleSubmitted:
		return decodeOracleSubmitted(log)
	case topicMarketResolved:
		return decodeMarketResolved(log)
	case topicRedeemed:
		return decodeRedeemed(log)
	default:
		return DecodedEvent{}, fmt.Errorf("unknown event topic %s", log.Topics[0].Hex())
	}
}

func decodeMarketCreated(log types.Log) (DecodedEvent, error) {
	if len(log.Topics) != 4 {
		return DecodedEvent{}, errors.New("invalid MarketCreated topic count")
	}
	values, err := marketCreatedArgs.Unpack(log.Data)
	if err != nil {
		return DecodedEvent{}, err
	}
	lockTime := values[0].(*big.Int)

	return event(EventMarketCreated, MarketCreatedPayload{
		MarketID:       topicBig(log.Topics[1]).String(),
		Market:         topicAddress(log.Topics[2]).Hex(),
		Creator:        topicAddress(log.Topics[3]).Hex(),
		LockTime:       lockTime.Uint64(),
		ResolutionRule: values[1].(string),
	})
}

func decodeTradeExecuted(log types.Log) (DecodedEvent, error) {
	if len(log.Topics) != 4 {
		return DecodedEvent{}, errors.New("invalid TradeExecuted topic count")
	}
	values, err := uintArg.Unpack(log.Data)
	if err != nil {
		return DecodedEvent{}, err
	}
	return event(EventTradeExecuted, TradeExecutedPayload{
		MarketID: topicBig(log.Topics[1]).String(),
		User:     topicAddress(log.Topics[2]).Hex(),
		Side:     outcomeString(uint8(topicBig(log.Topics[3]).Uint64())),
		Amount:   values[0].(*big.Int).String(),
	})
}

func decodeMarketLocked(log types.Log) (DecodedEvent, error) {
	if len(log.Topics) != 2 {
		return DecodedEvent{}, errors.New("invalid MarketLocked topic count")
	}
	values, err := uintArg.Unpack(log.Data)
	if err != nil {
		return DecodedEvent{}, err
	}
	return event(EventMarketLocked, MarketLockedPayload{
		MarketID: topicBig(log.Topics[1]).String(),
		LockTime: values[0].(*big.Int).Uint64(),
	})
}

func decodeOracleSubmitted(log types.Log) (DecodedEvent, error) {
	if len(log.Topics) != 4 {
		return DecodedEvent{}, errors.New("invalid OracleSubmitted topic count")
	}
	values, err := uintArg.Unpack(log.Data)
	if err != nil {
		return DecodedEvent{}, err
	}
	return event(EventOracleSubmitted, OracleSubmittedPayload{
		MarketID: topicBig(log.Topics[1]).String(),
		Outcome:  outcomeString(uint8(topicBig(log.Topics[2]).Uint64())),
		Nonce:    topicBig(log.Topics[3]).String(),
		Expiry:   values[0].(*big.Int).Uint64(),
	})
}

func decodeMarketResolved(log types.Log) (DecodedEvent, error) {
	if len(log.Topics) != 3 {
		return DecodedEvent{}, errors.New("invalid MarketResolved topic count")
	}
	return event(EventMarketResolved, MarketResolvedPayload{
		MarketID: topicBig(log.Topics[1]).String(),
		Outcome:  outcomeString(uint8(topicBig(log.Topics[2]).Uint64())),
	})
}

func decodeRedeemed(log types.Log) (DecodedEvent, error) {
	if len(log.Topics) != 3 {
		return DecodedEvent{}, errors.New("invalid Redeemed topic count")
	}
	values, err := uintArg.Unpack(log.Data)
	if err != nil {
		return DecodedEvent{}, err
	}
	return event(EventRedeemed, RedeemedPayload{
		MarketID: topicBig(log.Topics[1]).String(),
		User:     topicAddress(log.Topics[2]).Hex(),
		Amount:   values[0].(*big.Int).String(),
	})
}

func event(kind string, payload any) (DecodedEvent, error) {
	encoded, err := json.Marshal(payload)
	if err != nil {
		return DecodedEvent{}, err
	}
	return DecodedEvent{Type: kind, Payload: encoded}, nil
}

func topicBig(topic common.Hash) *big.Int {
	return new(big.Int).SetBytes(topic.Bytes())
}

func topicAddress(topic common.Hash) common.Address {
	return common.BytesToAddress(topic.Bytes()[12:])
}

func outcomeString(outcome uint8) string {
	switch outcome {
	case 1:
		return "YES"
	case 2:
		return "NO"
	default:
		return fmt.Sprintf("UNKNOWN_%d", outcome)
	}
}

func mustABIType(raw string) abi.Type {
	typ, err := abi.NewType(raw, "", nil)
	if err != nil {
		panic(err)
	}
	return typ
}

func EventTypeFromString(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}

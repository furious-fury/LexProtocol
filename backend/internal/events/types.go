package events

import sharedtypes "github.com/lexprotocol/lexprotocol/shared/types"

type (
	Event      = sharedtypes.Event
	TradeEvent = sharedtypes.TradeEvent
	TradeSide  = sharedtypes.TradeSide
)

const EventTradeExecuted = sharedtypes.EventTradeExecuted

package dataflows

// TradingContextJSON represents the complete trading context sent to LLM
// TradingContextJSON 表示发送给 LLM 的完整交易上下文
type TradingContextJSON struct {
	Instructions string                       `json:"instructions"` // Trading instructions from prompt file
	Schema       *TradingOutputSchema         `json:"schema"`       // Expected output schema
	Account      *AccountDataJSON             `json:"account"`      // Account information
	Positions    map[string]*PositionDataJSON `json:"positions"`    // Current positions by symbol
	Market       map[string]*SymbolMarketData `json:"market"`       // Market data by symbol
}

// AccountDataJSON represents account information in JSON format
// AccountDataJSON 表示 JSON 格式的账户信息
type AccountDataJSON struct {
	TotalBalance       float64 `json:"total_balance"`        // Total account balance in USDT
	AvailableBalance   float64 `json:"available_balance"`    // Available balance for trading
	TotalUnrealizedPnL float64 `json:"total_unrealized_pnl"` // Total unrealized PnL
	TotalMarginUsed    float64 `json:"total_margin_used"`    // Total margin used
	MarginRatio        float64 `json:"margin_ratio"`         // Margin ratio (used/total)
}

// PositionDataJSON represents a single position in JSON format
// PositionDataJSON 表示 JSON 格式的单个持仓
type PositionDataJSON struct {
	Symbol           string  `json:"symbol"`             // Trading pair (e.g., "BTC/USDT")
	Side             string  `json:"side"`               // Position side: "LONG" or "SHORT"
	EntryPrice       float64 `json:"entry_price"`        // Entry price
	CurrentPrice     float64 `json:"current_price"`      // Current market price
	PositionSize     float64 `json:"position_size"`      // Position size in base currency
	Leverage         int     `json:"leverage"`           // Leverage used
	UnrealizedPnL    float64 `json:"unrealized_pnl"`     // Unrealized PnL in USDT
	UnrealizedPnLPct float64 `json:"unrealized_pnl_pct"` // Unrealized PnL percentage
	StopLoss         float64 `json:"stop_loss"`          // Stop loss price
	TakeProfit       float64 `json:"take_profit"`        // Take profit price
	MarginUsed       float64 `json:"margin_used"`        // Margin used for this position
	HighestPrice     float64 `json:"highest_price"`      // Highest price since entry (for trailing stop)
	LowestPrice      float64 `json:"lowest_price"`       // Lowest price since entry (for trailing stop)
}

// TradingOutputSchema represents the expected output schema for LLM
// TradingOutputSchema 表示 LLM 的预期输出 Schema
type TradingOutputSchema struct {
	Type       string                        `json:"type"`
	Properties map[string]*TradingPairSchema `json:"properties"`
	Required   []string                      `json:"required,omitempty"`
}

// TradingPairSchema represents the schema for a single trading pair output
// TradingPairSchema 表示单个交易对输出的 Schema
type TradingPairSchema struct {
	Type       string                     `json:"type"`
	Properties map[string]*PropertySchema `json:"properties"`
	Required   []string                   `json:"required,omitempty"`
}

// PropertySchema represents a property in the schema
// PropertySchema 表示 Schema 中的一个属性
type PropertySchema struct {
	Type        string      `json:"type,omitempty"`
	Description string      `json:"description,omitempty"`
	Enum        []string    `json:"enum,omitempty"`
	Minimum     *float64    `json:"minimum,omitempty"`
	Maximum     *float64    `json:"maximum,omitempty"`
	Properties  interface{} `json:"properties,omitempty"`
	Items       interface{} `json:"items,omitempty"`
}

// GetTradingOutputSchema returns the expected output schema for LLM
// GetTradingOutputSchema 返回 LLM 的预期输出 Schema
func GetTradingOutputSchema() *TradingOutputSchema {
	minConfidence := 0.0
	maxConfidence := 1.0
	minLeverage := 5.0
	maxLeverage := 15.0
	minRiskReward := 3.5

	return &TradingOutputSchema{
		Type: "object",
		Properties: map[string]*TradingPairSchema{
			"<SYMBOL>": {
				Type: "object",
				Properties: map[string]*PropertySchema{
					"trend_analyzer": {
						Type: "object",
						Properties: map[string]*PropertySchema{
							"trend": {
								Type:        "string",
								Enum:        []string{"UP", "DOWN", "SIDEWAYS"},
								Description: "Market trend direction",
							},
							"phase": {
								Type:        "string",
								Enum:        []string{"PULLBACK", "IMPULSE", "CORRECTION", "CONSOLIDATION"},
								Description: "Current market phase",
							},
							"risk": {
								Type:        "string",
								Enum:        []string{"LOW", "MID", "HIGH"},
								Description: "Risk level assessment",
							},
						},
					},
					"confidence_to_leverage": {
						Type: "object",
						Properties: map[string]*PropertySchema{
							"confidence": {
								Type:        "number",
								Minimum:     &minConfidence,
								Maximum:     &maxConfidence,
								Description: "Confidence level (0.0-1.0)",
							},
							"leverage": {
								Type:        "number",
								Minimum:     &minLeverage,
								Maximum:     &maxLeverage,
								Description: "Leverage to use (5-15)",
							},
						},
					},
					"risk_metrics": {
						Type: "object",
						Properties: map[string]*PropertySchema{
							"ATR": {
								Type:        "number",
								Description: "Average True Range value",
							},
							"stop_loss": {
								Type:        "number",
								Description: "Stop loss price",
							},
							"take_profit": {
								Type:        "number",
								Description: "Take profit price",
							},
							"estimated_risk_reward": {
								Type:        "number",
								Minimum:     &minRiskReward,
								Description: "Estimated risk/reward ratio (minimum 3.5)",
							},
							"support_levels": {
								Type:        "array",
								Description: "Support price levels",
							},
							"resistance_levels": {
								Type:        "array",
								Description: "Resistance price levels",
							},
						},
					},
					"decision_gate": {
						Type: "object",
						Properties: map[string]*PropertySchema{
							"value": {
								Type:        "string",
								Enum:        []string{"NO_TRADE", "TRADE"},
								Description: "Trading decision gate",
							},
							"reason_code": {
								Type: "array",
								Items: map[string]interface{}{
									"type": "string",
									"enum": []string{"RISK_REWARD_LOW", "OVERBOUGHT", "OVERSOLD", "VOLATILITY_HIGH", "MARKET_UNCLEAR", "TREND_WEAK"},
								},
								Description: "Reasons for NO_TRADE decision",
							},
						},
					},
					"trading_signal": {
						Type: "object",
						Properties: map[string]*PropertySchema{
							"action": {
								Type:        "string",
								Enum:        []string{"BUY", "SELL", "HOLD"},
								Description: "Trading action",
							},
							"stop_loss": {
								Type:        "number",
								Description: "Stop loss price (0 if HOLD)",
							},
							"take_profit": {
								Type:        "number",
								Description: "Take profit price (0 if HOLD)",
							},
							"position_size": {
								Type:        "number",
								Description: "Position size percentage (0-100, 0 if HOLD)",
							},
							"reasoning": {
								Type:        "string",
								Description: "Brief reasoning for the decision",
							},
						},
					},
					"position_adjustment": {
						Type: "object",
						Properties: map[string]*PropertySchema{
							"action": {
								Type:        "string",
								Enum:        []string{"CLOSE_LONG", "CLOSE_SHORT", "HOLD"},
								Description: "Position adjustment action (only for existing positions)",
							},
							"stop_loss_adjustment": {
								Type:        "string",
								Enum:        []string{"ALLOW", "HOLD", "DISALLOW"},
								Description: "Stop loss adjustment strategy: ALLOW=calculate new stop based on structure, HOLD=keep current stop, DISALLOW=prohibit adjustment",
							},
							"take_profit_adjustment": {
								Type:        "number",
								Description: "New take profit price (null if no adjustment)",
							},
							"reasoning": {
								Type:        "string",
								Description: "Reasoning for position adjustment",
							},
						},
					},
				},
			},
		},
	}
}

// BuildTradingContextJSON builds the complete trading context JSON for LLM
// BuildTradingContextJSON 构建发送给 LLM 的完整交易上下文 JSON
func BuildTradingContextJSON(
	instructions string,
	accountInfo string,
	positionsInfo string,
	marketData map[string]*SymbolMarketData,
) *TradingContextJSON {
	// Parse account info (simplified - you may need to enhance this)
	// 解析账户信息（简化版 - 可能需要增强）
	account := &AccountDataJSON{
		TotalBalance:       0,
		AvailableBalance:   0,
		TotalUnrealizedPnL: 0,
		TotalMarginUsed:    0,
		MarginRatio:        0,
	}

	// Parse positions info (simplified - you may need to enhance this)
	// 解析持仓信息（简化版 - 可能需要增强）
	positions := make(map[string]*PositionDataJSON)

	return &TradingContextJSON{
		Instructions: instructions,
		Schema:       GetTradingOutputSchema(),
		Account:      account,
		Positions:    positions,
		Market:       marketData,
	}
}

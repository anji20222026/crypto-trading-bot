package agents

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	openaiComponent "github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/eino-contrib/jsonschema"
	"github.com/oak/crypto-trading-bot/internal/config"
	"github.com/oak/crypto-trading-bot/internal/dataflows"
	"github.com/oak/crypto-trading-bot/internal/executors"
	"github.com/oak/crypto-trading-bot/internal/logger"
)

// SymbolReports holds reports for a single symbol
// SymbolReports 保存单个交易对的报告
type SymbolReports struct {
	Symbol                    string
	MarketReport              string
	CryptoReport              string
	SentimentReport           string
	PositionInfo              string
	OHLCVData                 []dataflows.OHLCV
	TechnicalIndicators       *dataflows.TechnicalIndicators // 主时间周期的技术指标 / Primary timeframe indicators
	LongerTechnicalIndicators *dataflows.TechnicalIndicators // 长期时间周期的技术指标 / Longer timeframe indicators
	LongerOHLCVData           []dataflows.OHLCV              // 长期时间周期的 OHLCV 数据 / Longer timeframe OHLCV data
	MarketJSONData            *dataflows.SymbolMarketData    // 结构化市场数据 / Structured market data for LLM
}

// TradeDecision represents a structured trading decision from LLM (for JSON Schema output)
// TradeDecision 表示 LLM 的结构化交易决策（用于 JSON Schema 输出）
type TradeDecision struct {
	Symbol                     string   `json:"symbol"`                                  // 交易对 / Trading pair
	Action                     string   `json:"action"`                                  // 交易动作 / Action: BUY|SELL|HOLD|CLOSE_LONG|CLOSE_SHORT
	Confidence                 float64  `json:"confidence"`                              // 置信度 / Confidence (0.00-1.00)
	Leverage                   int      `json:"leverage"`                                // 杠杆倍数 / Leverage multiplier
	PositionSize               float64  `json:"position_size"`                           // 建议仓位百分比 / Position size percentage (0-100)
	StopLoss                   float64  `json:"stop_loss"`                               // 止损价格 / Stop loss price
	Reasoning                  string   `json:"reasoning"`                               // 交易理由 / Trading reasoning
	RiskRewardRatio            float64  `json:"risk_reward_ratio"`                       // 预期盈亏比 / Risk/reward ratio
	Summary                    string   `json:"summary"`                                 // 总结 / Summary
	CurrentPnlPercent          *float64 `json:"current_pnl_percent,omitempty"`           // 当前盈亏% (仅HOLD) / Current PnL% (HOLD only)
	NewStopLoss                *float64 `json:"new_stop_loss,omitempty"`                 // 新止损价格 (仅HOLD调整时) / New stop loss (HOLD adjustment only)
	StopLossReason             *string  `json:"stop_loss_reason,omitempty"`              // 止损调整理由 (仅HOLD调整时) / Stop loss reason (HOLD adjustment only)
	StopLossAdjustmentStrategy *string  `json:"stop_loss_adjustment_strategy,omitempty"` // 止损调整策略 (ALLOW/HOLD/DISALLOW) / Stop loss adjustment strategy
}

// TradingPairsResponse represents the new response format with trading_pairs structure
// TradingPairsResponse 表示新的 trading_pairs 结构响应格式
type TradingPairsResponse struct {
	TradingPairs map[string]*SymbolDecision `json:"trading_pairs"` // 交易对决策映射 / Trading pairs decision map
}

// SymbolDecision represents a single symbol's complete decision structure
// SymbolDecision 表示单个交易对的完整决策结构
type SymbolDecision struct {
	TrendAnalyzer        *TrendAnalyzer        `json:"trend_analyzer"`                // 趋势分析 / Trend analysis
	ConfidenceToLeverage *ConfidenceToLeverage `json:"confidence_to_leverage"`        // 置信度与杠杆 / Confidence and leverage
	RiskMetrics          *RiskMetrics          `json:"risk_metrics"`                  // 风险指标 / Risk metrics
	DecisionGate         *DecisionGate         `json:"decision_gate"`                 // 决策门 / Decision gate
	TradingSignal        *TradingSignal        `json:"trading_signal"`                // 交易信号 / Trading signal
	PositionAdjustment   *PositionAdjustment   `json:"position_adjustment,omitempty"` // 仓位调整 / Position adjustment
}

// TrendAnalyzer represents trend analysis data
// TrendAnalyzer 表示趋势分析数据
type TrendAnalyzer struct {
	Trend *EnumValue `json:"trend"` // 趋势方向 / Trend direction
	Phase *EnumValue `json:"phase"` // 趋势阶段 / Trend phase
	Risk  *EnumValue `json:"risk"`  // 风险等级 / Risk level
}

// ConfidenceToLeverage represents confidence and leverage mapping
// ConfidenceToLeverage 表示置信度与杠杆映射
type ConfidenceToLeverage struct {
	Confidence *ValueOnly     `json:"confidence"` // 置信度 / Confidence
	Leverage   *LeverageValue `json:"leverage"`   // 杠杆 / Leverage
}

// RiskMetrics represents risk-related metrics
// RiskMetrics 表示风险相关指标
type RiskMetrics struct {
	ATR                 *ATRValue  `json:"ATR"`                   // ATR 指标 / ATR indicator
	StopLoss            *ValueOnly `json:"stop_loss"`             // 止损价格 / Stop loss price
	TakeProfit          *ValueOnly `json:"take_profit"`           // 止盈价格 / Take profit price
	EstimatedRiskReward *ValueOnly `json:"estimated_risk_reward"` // 预估盈亏比 / Estimated risk/reward
	SupportLevels       []float64  `json:"support_levels"`        // 支撑位 / Support levels
	ResistanceLevels    []float64  `json:"resistance_levels"`     // 阻力位 / Resistance levels
}

// DecisionGate represents the decision gate
// DecisionGate 表示决策门
type DecisionGate struct {
	Value      string      `json:"value"`       // 决策值 / Decision value
	ReasonCode *ReasonCode `json:"reason_code"` // 原因代码 / Reason code
}

// TradingSignal represents the trading signal
// TradingSignal 表示交易信号
type TradingSignal struct {
	Action       *EnumValue `json:"action"`        // 交易动作 / Trading action
	StopLoss     float64    `json:"stop_loss"`     // 止损价格 / Stop loss price
	TakeProfit   float64    `json:"take_profit"`   // 止盈价格 / Take profit price
	PositionSize float64    `json:"position_size"` // 仓位大小 / Position size
	Reasoning    string     `json:"reasoning"`     // 理由 / Reasoning
}

// PositionAdjustment represents position adjustment data
// PositionAdjustment 表示仓位调整数据
type PositionAdjustment struct {
	Action               *EnumValue `json:"action"`                 // 调整动作 / Adjustment action
	StopLossAdjustment   *EnumValue `json:"stop_loss_adjustment"`   // 止损调整策略 / Stop loss adjustment strategy
	TakeProfitAdjustment *ValueOnly `json:"take_profit_adjustment"` // 止盈调整 / Take profit adjustment
	Reasoning            string     `json:"reasoning"`              // 理由 / Reasoning
}

// StopLossAdjustment strategy values:
// 止损调整策略值：
//   - ALLOW: Allow calculating new stop-loss based on new support/resistance
//     允许基于新的支撑/阻力计算更紧的止损
//   - HOLD: Keep current stop-loss, do not calculate new one (technically possible but not this time)
//     保持当前止损，不计算新止损（技术上可以，但这次不动）
//   - DISALLOW: Prohibit any stop-loss adjustment (e.g., during high volatility)
//     禁止任何止损调整（策略级禁止，例如波动异常时）

// EnumValue represents a value with enum constraint
// EnumValue 表示带枚举约束的值
type EnumValue struct {
	Value string `json:"value"` // 值 / Value
}

// ValueOnly represents a simple value wrapper
// ValueOnly 表示简单值包装器
type ValueOnly struct {
	Value float64 `json:"value"` // 值 / Value
}

// LeverageValue represents leverage with min/max constraints
// LeverageValue 表示带最小/最大约束的杠杆值
type LeverageValue struct {
	Value int `json:"value"` // 值 / Value
	Min   int `json:"min"`   // 最小值 / Minimum value
	Max   int `json:"max"`   // 最大值 / Maximum value
}

// ATRValue represents ATR indicator data
// ATRValue 表示 ATR 指标数据
type ATRValue struct {
	Value float64 `json:"value"` // 值 / Value
}

// ReasonCode represents reason codes for decision gate
// ReasonCode 表示决策门的原因代码
type ReasonCode struct {
	Value []string `json:"value"` // 原因代码列表 / Reason code list
}

// UnmarshalJSON implements custom JSON unmarshaling for ReasonCode
// UnmarshalJSON 实现 ReasonCode 的自定义 JSON 解析
func (rc *ReasonCode) UnmarshalJSON(data []byte) error {
	// Try to unmarshal as {"value": [...]} format first
	// 首先尝试解析为 {"value": [...]} 格式
	type Alias ReasonCode
	aux := &struct {
		*Alias
	}{
		Alias: (*Alias)(rc),
	}
	if err := sonic.Unmarshal(data, &aux); err == nil && rc.Value != nil {
		return nil
	}

	// Fallback: try to unmarshal as direct array [...] format
	// 回退：尝试解析为直接数组 [...] 格式
	var directArray []string
	if err := sonic.Unmarshal(data, &directArray); err == nil {
		rc.Value = directArray
		return nil
	}

	return fmt.Errorf("无法解析 reason_code: %s", string(data))
}

// AgentState holds the state of all analysts' reports for multiple symbols
// AgentState 保存所有分析师对多个交易对的报告状态
type AgentState struct {
	Symbols       []string                  // 所有交易对 / All trading pairs
	Timeframe     string                    // 时间周期 / Timeframe
	Reports       map[string]*SymbolReports // 每个交易对的报告 / Reports for each symbol
	AccountInfo   string                    // 账户总览信息 / Account overview
	AllPositions  string                    // 所有持仓汇总 / All positions summary
	FinalDecision string                    // 最终交易决策 / Final trading decision
	mu            sync.RWMutex              // 读写锁 / Read-write mutex
}

// NewAgentState creates a new agent state for multiple symbols
// NewAgentState 为多个交易对创建新的状态
func NewAgentState(symbols []string, timeframe string) *AgentState {
	reports := make(map[string]*SymbolReports)
	for _, symbol := range symbols {
		reports[symbol] = &SymbolReports{
			Symbol: symbol,
		}
	}
	return &AgentState{
		Symbols:   symbols,
		Timeframe: timeframe,
		Reports:   reports,
	}
}

// SetMarketReport sets the market analysis report for a symbol
// SetMarketReport 设置某个交易对的市场分析报告
func (s *AgentState) SetMarketReport(symbol, report string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, exists := s.Reports[symbol]; exists {
		r.MarketReport = report
	}
}

// SetCryptoReport sets the crypto analysis report for a symbol
// SetCryptoReport 设置某个交易对的加密货币分析报告
func (s *AgentState) SetCryptoReport(symbol, report string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, exists := s.Reports[symbol]; exists {
		r.CryptoReport = report
	}
}

// SetSentimentReport sets the sentiment analysis report for a symbol
// SetSentimentReport 设置某个交易对的情绪分析报告
func (s *AgentState) SetSentimentReport(symbol, report string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, exists := s.Reports[symbol]; exists {
		r.SentimentReport = report
	}
}

// SetPositionInfo sets the position information for a symbol
// SetPositionInfo 设置某个交易对的持仓信息
func (s *AgentState) SetPositionInfo(symbol, info string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, exists := s.Reports[symbol]; exists {
		r.PositionInfo = info
	}
}

// SetAccountInfo sets the account overview information
// SetAccountInfo 设置账户总览信息
func (s *AgentState) SetAccountInfo(info string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AccountInfo = info
}

// SetAllPositions sets the all positions summary
// SetAllPositions 设置所有持仓汇总
func (s *AgentState) SetAllPositions(info string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.AllPositions = info
}

// SetFinalDecision sets the final trading decision
// SetFinalDecision 设置最终交易决策
func (s *AgentState) SetFinalDecision(decision string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.FinalDecision = decision
}

// GetSymbolReports returns reports for a specific symbol
// GetSymbolReports 返回特定交易对的报告
func (s *AgentState) GetSymbolReports(symbol string) *SymbolReports {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.Reports[symbol]
}

// GetAllReports returns all reports as a formatted string
// GetAllReports 返回所有报告的格式化字符串
func (s *AgentState) GetAllReports() string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var sb strings.Builder

	// 首先显示账户总览 / First show account overview
	if s.AccountInfo != "" {
		sb.WriteString("\n=== 账户总览 ===\n")
		sb.WriteString(s.AccountInfo)
		sb.WriteString("\n")
	}

	// 然后显示所有持仓汇总 / Then show all positions summary
	if s.AllPositions != "" {
		sb.WriteString("=== 持仓汇总 ===\n")
		sb.WriteString(s.AllPositions)
		sb.WriteString("\n")
	}

	// 最后为每个交易对生成市场分析报告（不包含持仓信息）/ Finally generate market analysis for each symbol (without position info)
	for _, symbol := range s.Symbols {
		reports := s.Reports[symbol]
		sb.WriteString(fmt.Sprintf("\n================ %s 分析报告 ================\n", symbol))
		sb.WriteString("\n=== 市场技术分析 ===\n")
		sb.WriteString(reports.MarketReport)
		sb.WriteString("\n\n=== 加密货币专属分析 ===\n")
		sb.WriteString(reports.CryptoReport)
		//sb.WriteString("\n\n=== 市场情绪分析 ===\n")
		//sb.WriteString(reports.SentimentReport)
		sb.WriteString("\n")
	}

	return sb.String()
}

// loadPromptFromFile loads trading prompt from file, returns default prompt if file not found or error
// loadPromptFromFile 从文件加载交易策略 Prompt，如果文件不存在或出错则返回默认 Prompt
func loadPromptFromFile(promptPath string, log *logger.ColorLogger) string {
	// Default prompt - fallback if file not found
	// 默认 Prompt - 文件未找到时的后备方案
	defaultPrompt := `你是一位经验丰富的加密货币趋势交易员，遵循以下核心交易哲学：

**交易哲学**：
1. **极度选择性** - 只交易最确定的机会，宁可错过不可做错
2. **高盈亏比** - 目标盈亏比 ≥ 2:1，追求大赢
3. **快速止损** - 错了就认，绝不扛单
4. **让盈利奔跑** - 不设固定止盈，用追踪止损捕捉大行情
5. **耐心等待** - 等待高概率机会，做对的事比做很多事重要
6. **一次大赢胜过十次小赢** - 专注捕捉趋势性大行情

**决策原则**：
• 只在**强趋势**中交易（ADX > 25，趋势越强越好）
• 等待**趋势确认**（MACD、DI+/DI-、价格结构一致）
• 避免**追涨杀跌**（RSI 极端时谨慎，等待回调或突破）
• 要求**成交量配合**（放量突破更可靠）
• 从所有交易对中选择 **1-2 个最佳机会**，避免过度分散
• 大部分时候应该 **HOLD**，耐心等待完美设置

**决策输出格式**（必须严格遵守）：

【交易对名称】
**交易方向**: BUY / SELL / CLOSE_LONG / CLOSE_SHORT / HOLD
**置信度**: 0-1 的数值（只有 ≥ 0.75 才考虑交易）
**入场理由**: 为什么这是高确定性机会？（1-2 句话，说明趋势+确认信号）
**初始止损**: $具体价格（基于支撑/阻力或 2×ATR，必须输出数字）
**预期盈亏比**: ≥ 2:1（说明止损空间 vs 目标空间，但不设固定止盈）
**仓位建议**: 如 "30% 资金" 或 "维持观望"

**止损设置要求**（Critical）：
• 必须输出具体止损价格，如 "初始止损: $95000"
• 优先使用技术位（支撑/阻力）
• 次选 ATR：入场价 ± 2×ATR
• 底线：2-3% 固定止损
• 确保盈亏比：假设捕捉 5-10% 趋势，止损 2-3%，盈亏比 > 2:1

**重要提醒**：
⚠️ 只在极度确定（置信度 ≥ 0.75）时才交易，大部分时候应该 HOLD
⚠️ 不要设置固定止盈 - 我们用追踪止损让盈利奔跑
⚠️ 一次 10% 大赢比十次 1% 小赢更重要
⚠️ 宁可错过 100 次机会，也不做 1 次不确定的交易

---

最后包含总结：说明为什么选择这些交易对，整体盈亏比如何，风险如何控制。

请用中文回答，语言简洁专业。`

	// Try to read from file
	// 尝试从文件读取
	if promptPath == "" {
		log.Warning("Prompt 文件路径为空，使用默认 Prompt")
		return defaultPrompt
	}

	content, err := os.ReadFile(promptPath)
	if err != nil {
		log.Warning(fmt.Sprintf("无法读取 Prompt 文件 %s: %v，使用默认 Prompt", promptPath, err))
		return defaultPrompt
	}

	promptContent := strings.TrimSpace(string(content))
	if promptContent == "" {
		log.Warning(fmt.Sprintf("Prompt 文件 %s 为空，使用默认 Prompt", promptPath))
		return defaultPrompt
	}

	log.Success(fmt.Sprintf("成功加载交易策略 Prompt: %s", promptPath))
	return promptContent
}

// SimpleTradingGraph creates a simplified trading workflow using Eino Graph
type SimpleTradingGraph struct {
	config          *config.Config
	logger          *logger.ColorLogger
	executor        *executors.BinanceExecutor
	state           *AgentState
	stopLossManager *executors.StopLossManager
	startTime       time.Time  // 交易开始时间 / Trading start time
	tradeCount      int        // 已执行的交易次数 / Number of trades executed
	mu              sync.Mutex // 保护 tradeCount / Protect tradeCount
}

// NewSimpleTradingGraph creates a new simple trading graph
// NewSimpleTradingGraph 创建新的简单交易图
func NewSimpleTradingGraph(cfg *config.Config, log *logger.ColorLogger, executor *executors.BinanceExecutor, stopLossManager *executors.StopLossManager) *SimpleTradingGraph {
	return &SimpleTradingGraph{
		config:          cfg,
		logger:          log,
		executor:        executor,
		state:           NewAgentState(cfg.CryptoSymbols, cfg.CryptoTimeframe),
		stopLossManager: stopLossManager,
		startTime:       time.Now(), // 初始化交易开始时间 / Initialize trading start time
		tradeCount:      0,          // 初始化交易次数为 0 / Initialize trade count to 0
	}
}

// IncrementTradeCount increments the trade counter (thread-safe)
// IncrementTradeCount 增加交易计数（线程安全）
func (g *SimpleTradingGraph) IncrementTradeCount() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.tradeCount++
}

// GetTradeCount returns the current trade count (thread-safe)
// GetTradeCount 返回当前交易次数（线程安全）
func (g *SimpleTradingGraph) GetTradeCount() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.tradeCount
}

// BuildGraph constructs the trading workflow graph with parallel execution
func (g *SimpleTradingGraph) BuildGraph(ctx context.Context) (compose.Runnable[map[string]any, map[string]any], error) {
	graph := compose.NewGraph[map[string]any, map[string]any]()

	marketData := dataflows.NewMarketData(g.config)

	// Market Analyst Lambda - Fetches market data and calculates indicators for all symbols
	// Market Analyst Lambda - 为所有交易对获取市场数据并计算指标
	marketAnalyst := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		g.logger.Info("🔍 市场分析师：正在获取所有交易对的市场数据...")

		timeframe := g.config.CryptoTimeframe
		lookbackDays := g.config.CryptoLookbackDays

		// 并行分析所有交易对 / Analyze all symbols in parallel
		var wg sync.WaitGroup
		var mu sync.Mutex
		results := make(map[string]any)

		for _, symbol := range g.state.Symbols {
			wg.Add(1)
			go func(sym string) {
				defer wg.Done()

				g.logger.Info(fmt.Sprintf("  📊 正在分析 %s...", sym))

				binanceSymbol := g.config.GetBinanceSymbolFor(sym)

				// Fetch OHLCV data for primary timeframe
				// 获取主时间周期的 OHLCV 数据
				ohlcvData, err := marketData.GetOHLCV(ctx, binanceSymbol, timeframe, lookbackDays)
				if err != nil {
					g.logger.Warning(fmt.Sprintf("  ⚠️  %s OHLCV数据获取失败: %v", sym, err))
					return
				}

				// Calculate indicators for primary timeframe
				// 计算主时间周期的指标
				indicators := dataflows.CalculateIndicators(ohlcvData)

				// Generate primary timeframe report
				// 生成主时间周期报告
				report := dataflows.FormatIndicatorReport(sym, timeframe, ohlcvData, indicators)

				// Multi-timeframe analysis (if enabled)
				// 多时间周期分析（如果启用）
				var longerIndicators *dataflows.TechnicalIndicators
				var longerOHLCV []dataflows.OHLCV
				if g.config.EnableMultiTimeframe {
					g.logger.Info(fmt.Sprintf("  🔄 正在获取 %s 更长期时间周期数据 (%s)...", sym, g.config.CryptoLongerTimeframe))

					// Fetch OHLCV data for longer timeframe
					// 获取更长期时间周期的 OHLCV 数据
					longerOHLCV, err = marketData.GetOHLCV(ctx, binanceSymbol, g.config.CryptoLongerTimeframe, g.config.CryptoLongerLookbackDays)
					if err != nil {
						g.logger.Warning(fmt.Sprintf("  ⚠️  %s 更长期时间周期数据获取失败: %v", sym, err))
					} else {
						// Calculate indicators for longer timeframe (with configurable ATR period for trailing stop)
						// 计算更长期时间周期的指标（使用可配置的 ATR 周期用于追踪止损）
						longerIndicators = dataflows.CalculateIndicators(longerOHLCV, g.config.TrailingStopATRPeriod)

						// Generate longer timeframe report
						// 生成更长期时间周期报告
						longerReport := dataflows.FormatLongerTimeframeReport(sym, g.config.CryptoLongerTimeframe, longerOHLCV, longerIndicators)

						// Append longer timeframe report to main report
						// 将更长期时间周期报告追加到主报告
						report += "\n" + longerReport

						g.logger.Success(fmt.Sprintf("  ✅ %s 多时间周期分析完成", sym))
					}
				}

				// Multi-timeframe indicators analysis (always enabled)
				// 多时间框架指标分析（默认启用）
				g.logger.Info(fmt.Sprintf("  📈 正在获取 %s 多时间框架指标...", sym))
				multiTimeframeIndicators := marketData.GetMultiTimeframeIndicators(ctx, binanceSymbol)
				if len(multiTimeframeIndicators) > 0 {
					multiTimeframeReport := dataflows.FormatMultiTimeframeReport(multiTimeframeIndicators)
					if multiTimeframeReport != "" {
						// Append multi-timeframe indicators report to main report
						// 将多时间框架指标报告追加到主报告
						report += "\n" + multiTimeframeReport
						g.logger.Success(fmt.Sprintf("  ✅ %s 多时间框架指标分析完成", sym))
					}
				}

				// Get current position info
				// 获取当前持仓信息
				var currentPosition *dataflows.CurrentPositionData
				if g.executor != nil {
					position, err := g.executor.GetCurrentPosition(ctx, sym)
					if err == nil && position != nil && position.Side != "" {
						// Convert position side to uppercase
						// 转换持仓方向为大写
						side := strings.ToUpper(position.Side)

						currentPosition = &dataflows.CurrentPositionData{
							Side:     side,
							StopLoss: position.CurrentStopLoss,
						}
						g.logger.Info(fmt.Sprintf("  📍 %s 当前持仓: %s, 止损: %.2f",
							sym, side, position.CurrentStopLoss))
					}
				}

				// Build structured JSON data for LLM
				// 为 LLM 构建结构化 JSON 数据
				g.logger.Info(fmt.Sprintf("  🔧 正在构建 %s 结构化市场数据...", sym))
				jsonData, err := dataflows.BuildMarketJSONData(
					ctx,
					marketData,
					binanceSymbol,
					timeframe,
					ohlcvData,
					indicators,
					longerIndicators,
					longerOHLCV,
					currentPosition, // Pass current position info / 传入当前持仓信息
				)
				if err != nil {
					g.logger.Warning(fmt.Sprintf("  ⚠️  %s 结构化数据构建失败: %v", sym, err))
				} else {
					g.logger.Success(fmt.Sprintf("  ✅ %s 结构化数据构建完成", sym))
				}

				// Save to state (thread-safe)
				mu.Lock()
				if reports := g.state.Reports[sym]; reports != nil {
					reports.OHLCVData = ohlcvData
					reports.TechnicalIndicators = indicators
					reports.LongerTechnicalIndicators = longerIndicators // 保存长期时间周期指标 / Save longer timeframe indicators
					reports.LongerOHLCVData = longerOHLCV                // 保存长期时间周期 OHLCV 数据 / Save longer timeframe OHLCV data
					reports.MarketJSONData = jsonData                    // 保存结构化市场数据 / Save structured market data
				}
				mu.Unlock()

				g.state.SetMarketReport(sym, report)

				g.logger.Success(fmt.Sprintf("  ✅ %s 市场分析完成", sym))
			}(symbol)
		}

		wg.Wait()
		g.logger.Success("✅ 所有交易对的市场分析完成")

		return results, nil
	})

	// Crypto Analyst Lambda - Fetches funding rate, order book, 24h stats for all symbols
	// Crypto Analyst Lambda - 为所有交易对获取资金费率、订单簿、24小时统计
	cryptoAnalyst := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		g.logger.Info("🔍 加密货币分析师：正在获取所有交易对的链上数据...")

		// 并行分析所有交易对 / Analyze all symbols in parallel
		var wg sync.WaitGroup
		results := make(map[string]any)

		for _, symbol := range g.state.Symbols {
			wg.Add(1)
			go func(sym string) {
				defer wg.Done()

				g.logger.Info(fmt.Sprintf("  🔗 正在分析 %s 链上数据...", sym))

				binanceSymbol := g.config.GetBinanceSymbolFor(sym)
				var reportBuilder strings.Builder

				reportBuilder.WriteString(fmt.Sprintf("=== %s 加密货币数据 ===\n\n", sym))

				// Funding rate
				fundingRate, err := marketData.GetFundingRate(ctx, binanceSymbol)
				if err != nil {
					reportBuilder.WriteString(fmt.Sprintf("资金费率获取失败: %v\n\n", err))
				} else {
					reportBuilder.WriteString(fmt.Sprintf("💰 资金费率: %.6f (%.4f%%)\n\n", fundingRate, fundingRate*100))
				}

				// Order book - use enhanced format
				//orderBook, err := marketData.GetOrderBook(ctx, binanceSymbol, 50)
				//if err != nil {
				//	reportBuilder.WriteString(fmt.Sprintf("订单簿获取失败: %v\n\n", err))
				//} else {
				//	// Use the new formatted order book report
				//	orderBookReport := dataflows.FormatOrderBookReport(orderBook, 20)
				//	reportBuilder.WriteString(orderBookReport)
				//	reportBuilder.WriteString("\n")
				//}

				// 持仓量统计 - 4h、15m 间隔，显示相对变化率
				// Open Interest Statistics - 4h window with 15m sampling, showing percentage changes
				reportBuilder.WriteString("📊 持仓量统计 (4h, 15m间隔):\n")
				reportBuilder.WriteString("注意：以下数据均为从旧到新，显示相对于上一个点的变化率\n")

				oiSeries, err := marketData.GetOpenInterestChange(ctx, binanceSymbol, "15m", 16)
				if err != nil {
					reportBuilder.WriteString(fmt.Sprintf("  数据获取失败: %v\n\n", err))
				} else if rawSeries, ok := oiSeries["series_values"].([]float64); ok && len(rawSeries) > 0 {
					// 显示起始值和结束值（绝对值）
					// Display start and end values (absolute values)

					// 计算相对于上一个点的百分比变化
					// Calculate percentage change relative to previous point
					parts := make([]string, 0, len(rawSeries))
					for i, val := range rawSeries {
						if i == 0 {
							// 第一个点作为基准
							// First point as baseline
							parts = append(parts, "0.00%")
						} else {
							previous := rawSeries[i-1]
							if previous > 0 {
								change := ((val - previous) / previous) * 100
								parts = append(parts, fmt.Sprintf("%+.2f%%", change))
							} else {
								parts = append(parts, "N/A")
							}
						}
					}
					reportBuilder.WriteString(fmt.Sprintf("持仓量变化率: [%s]\n", strings.Join(parts, ", ")))

					reportBuilder.WriteString("\n")
				} else {
					reportBuilder.WriteString("  数据不足，无法构建 4h 序列\n\n")
				}

				// 大户多空比 - 2h 15m 间隔，提供序列变化
				// Top Trader Long/Short Ratio - 2h window with 15m sampling
				//reportBuilder.WriteString("🐋 大户持仓多空比变化统计2h:\n")
				//
				//ratioSeries, err := marketData.GetTopLongShortPositionRatio(ctx, binanceSymbol, "15m", 8)
				//if err != nil {
				//	reportBuilder.WriteString(fmt.Sprintf("  数据获取失败: %v\n\n", err))
				//} else {
				//	longPct := ratioSeries["long_account"].(float64)
				//	shortPct := ratioSeries["short_account"].(float64)
				//	lsRatio := ratioSeries["long_short_ratio"].(float64)
				//	reportBuilder.WriteString(fmt.Sprintf("  最新: 多空比 %.2f (多头 %.1f%% vs 空头 %.1f%%)\n", lsRatio, longPct, shortPct))
				//
				//	if series, ok := ratioSeries["series_ratios"].([]float64); ok && len(series) > 0 {
				//		chunks := make([]string, 0, len(series))
				//		for _, val := range series {
				//			chunks = append(chunks, fmt.Sprintf("%.2f", val))
				//		}
				//		reportBuilder.WriteString(fmt.Sprintf("  间隔15分钟: [%s]\n\n", strings.Join(chunks, ", ")))
				//	} else {
				//		reportBuilder.WriteString("  数据不足，无法构建 2h 序列\n\n")
				//	}
				//}

				// 24h stats
				stats, err := marketData.Get24HrStats(ctx, binanceSymbol)
				if err != nil {
					reportBuilder.WriteString(fmt.Sprintf("📅 24h统计获取失败: %v\n", err))
				} else {
					reportBuilder.WriteString("📅 24h统计:\n")
					reportBuilder.WriteString(fmt.Sprintf("- 价格变化: %s%%, 最高: $%s, 最低: $%s, 成交量: %s\n",
						stats["price_change_percent"], stats["high_price"], stats["low_price"], stats["volume"]))
				}

				report := reportBuilder.String()
				g.state.SetCryptoReport(sym, report)

				g.logger.Success(fmt.Sprintf("  ✅ %s 加密货币分析完成", sym))
			}(symbol)
		}

		wg.Wait()
		g.logger.Success("✅ 所有交易对的加密货币分析完成")

		return results, nil
	})

	// Sentiment Analyst Lambda - Fetches market sentiment for all symbols
	// Sentiment Analyst Lambda - 为所有交易对获取市场情绪
	sentimentAnalyst := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		results := make(map[string]any)

		// Check if sentiment analysis is enabled
		// 检查是否启用情绪分析
		if !g.config.EnableSentimentAnalysis {
			g.logger.Info("ℹ️  市场情绪分析已禁用（ENABLE_SENTIMENT_ANALYSIS=false）")
			// Set empty sentiment reports for all symbols
			// 为所有交易对设置空的情绪报告
			for _, symbol := range g.state.Symbols {
				emptyReport := `
# 市场情绪分析（已禁用）

`
				g.state.SetSentimentReport(symbol, emptyReport)
			}
			return results, nil
		}

		g.logger.Info("🔍 情绪分析师：正在获取所有交易对的市场情绪...")

		// 并行分析所有交易对 / Analyze all symbols in parallel
		var wg sync.WaitGroup

		for _, symbol := range g.state.Symbols {
			wg.Add(1)
			go func(sym string) {
				defer wg.Done()

				g.logger.Info(fmt.Sprintf("  😊 正在分析 %s 市场情绪...", sym))

				// Extract base symbol (BTC from BTC/USDT)
				// 提取基础币种（从 BTC/USDT 提取 BTC）
				baseSymbol := strings.Split(sym, "/")[0]

				sentiment := dataflows.GetSentimentIndicators(ctx, baseSymbol)
				if sentiment == nil {
					g.logger.Warning(fmt.Sprintf("  ⚠️  %s 市场情绪数据获取失败", sym))
					report := dataflows.FormatSentimentReport(nil)
					g.state.SetSentimentReport(sym, report)
				} else {
					report := dataflows.FormatSentimentReport(sentiment)
					g.state.SetSentimentReport(sym, report)
					g.logger.Success(fmt.Sprintf("  ✅ %s 情绪分析完成", sym))
				}
			}(symbol)
		}

		wg.Wait()
		g.logger.Success("✅ 所有交易对的情绪分析完成")

		return results, nil
	})

	// Position Info Lambda - Gets current position for all symbols
	// Position Info Lambda - 获取所有交易对的持仓信息
	positionInfo := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		g.logger.Info("📊 获取账户总览和持仓信息...")

		// 首先获取账户信息（只调用一次）/ First get account info (call only once)
		accountSummary := g.executor.GetAccountSummary(ctx)
		g.state.SetAccountInfo(accountSummary)
		g.logger.Success("  ✅ 账户信息获取完成")

		// 并行获取所有交易对的持仓 / Get positions for all symbols in parallel
		var wg sync.WaitGroup
		results := make(map[string]any)
		positionSummaries := make(map[string]string) // 用于保存每个币种的持仓信息 / Store position info for each symbol
		var mu sync.Mutex                            // 保护 positionSummaries map

		for _, symbol := range g.state.Symbols {
			wg.Add(1)
			go func(sym string) {
				defer wg.Done()

				g.logger.Info(fmt.Sprintf("  📈 正在获取 %s 持仓...", sym))

				// Note: Trailing stop management is now handled by independent TrailingStopManager
				// 注意：追踪止损管理现在由独立的 TrailingStopManager 处理
				// This node only retrieves position status (read-only)
				// 此节点仅获取持仓状态（只读）

				// 获取持仓信息（不包含账户信息）/ Get position info (without account info)
				posInfo := g.executor.GetPositionOnly(ctx, sym, g.stopLossManager)

				mu.Lock()
				positionSummaries[sym] = posInfo
				mu.Unlock()

				g.logger.Success(fmt.Sprintf("  ✅ %s 持仓信息获取完成", sym))
			}(symbol)
		}

		wg.Wait()

		// 组合所有持仓信息 / Combine all position info
		var allPositions strings.Builder
		for _, symbol := range g.state.Symbols {
			allPositions.WriteString(fmt.Sprintf("**%s**:\n", symbol))
			allPositions.WriteString(positionSummaries[symbol])
			allPositions.WriteString("\n")
		}

		g.state.SetAllPositions(allPositions.String())
		g.logger.Success("✅ 账户总览和持仓信息获取完成")

		return results, nil
	})

	// Trader Lambda - Makes final decision using LLM
	trader := compose.InvokableLambda(func(ctx context.Context, input map[string]any) (map[string]any, error) {
		g.logger.Info("🤖 交易员：正在制定交易策略...")

		allReports := g.state.GetAllReports()

		// Try to use LLM for decision, fall back to simple rules if LLM fails
		var decision string
		var err error

		// Check if API key is configured
		if g.config.APIKey != "" && g.config.APIKey != "your_openai_key" {
			// ! Use LLM for decision
			decision, err = g.makeLLMDecision(ctx)
			if err != nil {
				g.logger.Warning(fmt.Sprintf("LLM 决策失败: %v", err))
				decision = g.makeSimpleDecision()
			}
		} else {
			g.logger.Info("OpenAI API Key 未配置，使用简单规则决策")
			decision = g.makeSimpleDecision()
		}

		g.state.SetFinalDecision(decision)

		g.logger.Decision(decision)

		return map[string]any{
			"decision":    decision,
			"all_reports": allReports,
		}, nil
	})

	// Add nodes to graph
	if err := graph.AddLambdaNode("market_analyst", marketAnalyst); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode("crypto_analyst", cryptoAnalyst); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode("sentiment_analyst", sentimentAnalyst); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode("position_info", positionInfo); err != nil {
		return nil, err
	}
	if err := graph.AddLambdaNode("trader", trader); err != nil {
		return nil, err
	}

	// Parallel execution: market_analyst and sentiment_analyst run in parallel
	if err := graph.AddEdge(compose.START, "market_analyst"); err != nil {
		return nil, err
	}
	if err := graph.AddEdge(compose.START, "sentiment_analyst"); err != nil {
		return nil, err
	}

	// After market_analyst completes, run crypto_analyst
	if err := graph.AddEdge("market_analyst", "crypto_analyst"); err != nil {
		return nil, err
	}

	// After crypto_analyst completes, get position info
	if err := graph.AddEdge("crypto_analyst", "position_info"); err != nil {
		return nil, err
	}

	// Wait for both sentiment_analyst and position_info before trader
	if err := graph.AddEdge("sentiment_analyst", "trader"); err != nil {
		return nil, err
	}
	if err := graph.AddEdge("position_info", "trader"); err != nil {
		return nil, err
	}

	// Trader outputs to END
	if err := graph.AddEdge("trader", compose.END); err != nil {
		return nil, err
	}

	// Compile with AllPredecessor trigger mode (wait for all inputs)
	return graph.Compile(ctx, compose.WithNodeTriggerMode(compose.AllPredecessor))
}

// makeSimpleDecision creates a simple rule-based decision (fallback when LLM is disabled)
// makeSimpleDecision 创建基于规则的简单决策（LLM 禁用时的后备方案）
func (g *SimpleTradingGraph) makeSimpleDecision() string {
	var decision strings.Builder

	decision.WriteString("=== 多币种交易决策分析 ===\n\n")
	decision.WriteString("说明: 这是基于规则的简单决策（LLM 未启用）。\n\n")

	// Analyze each symbol
	// 分析每个交易对
	for _, symbol := range g.state.Symbols {
		reports := g.state.GetSymbolReports(symbol)
		if reports == nil {
			continue
		}

		decision.WriteString(fmt.Sprintf("【%s】\n", symbol))

		// Analyze technical indicators if available
		// 如果有技术指标数据，进行分析
		if reports.TechnicalIndicators != nil && len(reports.OHLCVData) > 0 {
			lastIdx := len(reports.OHLCVData) - 1
			rsi := reports.TechnicalIndicators.RSI
			macd := reports.TechnicalIndicators.MACD
			signal := reports.TechnicalIndicators.Signal

			decision.WriteString("技术面分析:\n")

			// RSI analysis
			if len(rsi) > lastIdx {
				rsiVal := rsi[lastIdx]
				decision.WriteString(fmt.Sprintf("- RSI(14): %.2f ", rsiVal))
				if rsiVal > 70 {
					decision.WriteString("(超买区域，可能回调)\n")
				} else if rsiVal < 30 {
					decision.WriteString("(超卖区域，可能反弹)\n")
				} else {
					decision.WriteString("(中性区域)\n")
				}
			}

			// MACD analysis
			if len(macd) > lastIdx && len(signal) > lastIdx {
				macdVal := macd[lastIdx]
				signalVal := signal[lastIdx]
				decision.WriteString(fmt.Sprintf("- MACD: %.2f, Signal: %.2f ", macdVal, signalVal))
				if macdVal > signalVal {
					decision.WriteString("(MACD在Signal之上，多头信号)\n")
				} else {
					decision.WriteString("(MACD在Signal之下，空头信号)\n")
				}
			}
		}

		decision.WriteString(fmt.Sprintf("**建议**: HOLD（观望）\n\n"))
	}

	decision.WriteString("\n**最终决策**: HOLD（观望）\n")
	decision.WriteString("说明: 规则决策默认观望，建议启用 LLM 获得更智能的决策。\n")

	return decision.String()
}

// makeLLMDecision uses LLM to generate trading decision with JSON structured output
// makeLLMDecision 使用 LLM 生成交易决策，使用 JSON 结构化输出
func (g *SimpleTradingGraph) makeLLMDecision(ctx context.Context) (string, error) {
	// List of backend URLs that only support JSON Object mode (not JSON Schema)
	// 仅支持 JSON Object 模式（不支持 JSON Schema）的后端 URL 列表
	jsonObjectModeBackends := []string{
		"https://api.deepseek.com",                          // DeepSeek API
		"https://openrouter.ai/api/v1",                      // openrouter API
		"https://dashscope.aliyuncs.com/compatible-mode/v1", // Alibaba Cloud Qwen API

	}

	// Check if backend URL requires JSON Object mode
	// 检查后端 URL 是否需要 JSON Object 模式
	backendURL := strings.TrimSpace(g.config.BackendURL)
	backendURL = strings.TrimSuffix(backendURL, "/") // Remove trailing slash / 移除尾部斜杠

	useJSONObjectMode := false
	for _, backend := range jsonObjectModeBackends {
		backend = strings.TrimSuffix(backend, "/")
		if strings.HasPrefix(backendURL, backend) {
			useJSONObjectMode = true
			break
		}
	}

	var cfg *openaiComponent.ChatModelConfig

	if useJSONObjectMode {
		// Backends that only support JSON Object mode (no schema)
		// 仅支持 JSON Object 模式的后端（无 schema）
		g.logger.Info(fmt.Sprintf("检测到需要 JSON Object 模式的后端: %s", backendURL))
		cfg = &openaiComponent.ChatModelConfig{
			APIKey:  g.config.APIKey,
			BaseURL: g.config.BackendURL,
			Model:   g.config.QuickThinkLLM,
			// Enable basic JSON mode (compatible with DeepSeek, Qwen, etc.)
			// 启用基础 JSON 模式（兼容 DeepSeek、Qwen 等）
			ResponseFormat: &openaiComponent.ChatCompletionResponseFormat{
				Type: openaiComponent.ChatCompletionResponseFormatTypeJSONObject,
			},
		}
	} else {
		// OpenAI-compatible models: use JSON Schema mode
		// OpenAI 兼容模型：使用 JSON Schema 模式
		g.logger.Info("使用 OpenAI 兼容模式，启用 JSON Schema 多币种结构化输出")

		// Generate JSON Schema for multi-symbol trade decisions: map[symbol]TradeDecision
		// 使用反射为多币种决策生成 JSON Schema：map[交易对]TradeDecision
		var multiDecision map[string]TradeDecision
		jsonSchemaObj := jsonschema.Reflect(multiDecision)

		cfg = &openaiComponent.ChatModelConfig{
			APIKey:  g.config.APIKey,
			BaseURL: g.config.BackendURL,
			Model:   g.config.QuickThinkLLM,
			// Enable JSON Schema structured output
			// 启用 JSON Schema 结构化输出
			ResponseFormat: &openaiComponent.ChatCompletionResponseFormat{
				Type: openaiComponent.ChatCompletionResponseFormatTypeJSONSchema,
				JSONSchema: &openaiComponent.ChatCompletionResponseFormatJSONSchema{
					JSONSchema: jsonSchemaObj, // 使用 JSONSchema 字段而不是 Schema
					Strict:     false,         // eino-contrib/jsonschema 生成的 Schema 可能不完全兼容 strict 模式
				},
			},
		}
	}

	// Create ChatModel
	// 创建 ChatModel
	chatModel, err := openaiComponent.NewChatModel(ctx, cfg)
	if err != nil {
		g.logger.Warning(fmt.Sprintf("LLM 初始化失败，使用简单规则决策: %v", err))
		return g.makeSimpleDecision(), nil
	}

	// Build market data with schema
	// 构建带 Schema 的市场数据
	marketJSONData := &dataflows.MarketJSONData{
		Schema: dataflows.GetDefaultSchema(),
		Data:   make(map[string]*dataflows.SymbolMarketData),
	}

	g.state.mu.RLock()
	for _, symbol := range g.state.Symbols {
		if reports := g.state.Reports[symbol]; reports != nil && reports.MarketJSONData != nil {
			marketJSONData.Data[symbol] = reports.MarketJSONData
		}
	}
	g.state.mu.RUnlock()

	// Load system prompt from file (complete prompt with instructions and output schema)
	// 从文件加载系统 Prompt（包含完整的 instructions 和输出 schema）
	systemPrompt := loadPromptFromFile(g.config.TraderPromptPath, g.logger)

	// Build user prompt with market data schema and data
	// 构建用户 Prompt，包含市场数据 schema 和 data
	userDataJSON := map[string]interface{}{
		"schema": marketJSONData.Schema, // 市场数据的 schema（字段说明）
		"data":   marketJSONData.Data,   // 实际的市场数据
	}

	// Convert to JSON string
	// 转换为 JSON 字符串
	jsonBytes, err := sonic.MarshalIndent(userDataJSON, "", "  ")
	if err != nil {
		g.logger.Warning(fmt.Sprintf("JSON 序列化失败: %v，使用简单规则决策", err))
		return g.makeSimpleDecision(), nil
	}

	// User prompt is the market data with schema
	// 用户 Prompt 是带 schema 的市场数据
	userPrompt := string(jsonBytes)

	// Create messages
	// 创建消息
	messages := []*schema.Message{
		schema.SystemMessage(systemPrompt),
		schema.UserMessage(userPrompt),
	}

	// Log request details
	// 记录请求详情
	modeStr := "JSON Schema"
	if useJSONObjectMode {
		modeStr = "JSON Object"
	}
	g.logger.Header(fmt.Sprintf("📤 发送 LLM 请求 (%s 模式)", modeStr), '-', 80)
	g.logger.Info(fmt.Sprintf("模型: %s", g.config.QuickThinkLLM))
	g.logger.Info(fmt.Sprintf("后端: %s", g.config.BackendURL))
	g.logger.Info(fmt.Sprintf("系统提示词长度: %d 字符", len(systemPrompt)))
	g.logger.Info(fmt.Sprintf("用户数据长度: %d 字符", len(userPrompt)))

	// Count symbols in market data
	// 统计市场数据中的交易对数量
	symbolCount := len(marketJSONData.Data)
	g.logger.Info(fmt.Sprintf("包含交易对数量: %d", symbolCount))
	for symbol := range marketJSONData.Data {
		g.logger.Info(fmt.Sprintf("  - %s", symbol))
	}

	// Save complete request to LLM log file (not printed to console)
	// 保存完整请求到 LLM 日志文件（不打印到控制台）
	requestLog := fmt.Sprintf(`📤 完整 LLM 请求
================================================================================
时间: %s
模型: %s
模式: %s
后端: %s

系统提示词 (System Prompt):
--------------------------------------------------------------------------------
%s

用户数据 (User Prompt - Schema + Data):
--------------------------------------------------------------------------------
%s

================================================================================
`, time.Now().Format("2006-01-02_15-04-05"), g.config.QuickThinkLLM, modeStr, g.config.BackendURL, systemPrompt, userPrompt)

	// Use a counter to track request number (simple increment)
	// 使用计数器跟踪请求编号（简单递增）
	g.mu.Lock()
	requestNum := g.tradeCount + 1
	g.mu.Unlock()

	g.logger.SaveLLMRequest(requestLog, requestNum)

	// Call LLM
	// 调用 LLM
	g.logger.Info("🚀 正在调用 LLM...")
	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		g.logger.Warning(fmt.Sprintf("LLM 调用失败，使用简单规则决策: %v", err))
		return g.makeSimpleDecision(), nil
	}

	// Log response details
	// 记录响应详情
	g.logger.Header("📥 接收 LLM 响应", '-', 80)
	g.logger.Success("✅ LLM 决策生成完成")

	// Log token usage if available
	// 记录 token 使用情况
	var tokenInfo string
	if response.ResponseMeta != nil && response.ResponseMeta.Usage != nil {
		tokenInfo = fmt.Sprintf("Token 使用: %d (输入: %d, 输出: %d)",
			response.ResponseMeta.Usage.TotalTokens,
			response.ResponseMeta.Usage.PromptTokens,
			response.ResponseMeta.Usage.CompletionTokens)
		g.logger.Info(tokenInfo)
	}

	// Log response content length
	// 记录响应内容长度
	g.logger.Info(fmt.Sprintf("响应内容长度: %d 字符", len(response.Content)))

	// Save complete response to LLM log file (not printed to console)
	// 保存完整响应到 LLM 日志文件（不打印到控制台）
	responseLog := fmt.Sprintf(`📥 完整 LLM 响应
================================================================================
时间: %s
%s
响应长度: %d 字符

响应内容:
--------------------------------------------------------------------------------
%s

================================================================================
`, time.Now().Format("2006-01-02_15-04-05"), tokenInfo, len(response.Content), response.Content)

	g.logger.SaveLLMResponse(responseLog, requestNum)

	// Parse JSON response (support multiple formats)
	// 解析 JSON 响应（支持多种格式）
	g.logger.Header("🔍 解析 LLM 响应", '-', 80)

	cleanContent := extractJSONPayload(response.Content)
	trimmed := strings.TrimSpace(cleanContent)

	g.logger.Info(fmt.Sprintf("清理后的 JSON 长度: %d 字符", len(trimmed)))

	// Try to parse as direct map[string]*SymbolDecision format (without trading_pairs wrapper)
	// 尝试解析为直接的 map[string]*SymbolDecision 格式（不带 trading_pairs 外层）
	var directSymbolDecisions map[string]*SymbolDecision
	if err := sonic.Unmarshal([]byte(trimmed), &directSymbolDecisions); err == nil && len(directSymbolDecisions) > 0 {
		// Check if this is actually a SymbolDecision structure (has trend_analyzer, decision_gate, etc.)
		// 检查是否确实是 SymbolDecision 结构（包含 trend_analyzer、decision_gate 等）
		isSymbolDecisionFormat := false
		for _, decision := range directSymbolDecisions {
			if decision != nil && (decision.TrendAnalyzer != nil || decision.DecisionGate != nil || decision.TradingSignal != nil) {
				isSymbolDecisionFormat = true
				break
			}
		}

		if isSymbolDecisionFormat {
			g.logger.Success(fmt.Sprintf("✅ 成功解析为直接 SymbolDecision 格式（无 trading_pairs 外层），包含 %d 个交易对", len(directSymbolDecisions)))

			// Convert to legacy format for backward compatibility
			// 转换为旧格式以保持向后兼容
			convertedJSON, err := convertTradingPairsToLegacyFormat(directSymbolDecisions, g.logger)
			if err != nil {
				g.logger.Warning(fmt.Sprintf("转换直接 SymbolDecision 格式失败: %v", err))
				return g.makeSimpleDecision(), nil
			}

			// Log converted decisions
			// 记录转换后的决策
			for symbol, decision := range convertedJSON {
				g.logger.Info(fmt.Sprintf("  - %s: Action=%s, Confidence=%.2f, Leverage=%d",
					symbol, decision.Action, decision.Confidence, decision.Leverage))
			}

			// Convert back to JSON string for return
			// 转换回 JSON 字符串以返回
			convertedBytes, err := sonic.MarshalIndent(convertedJSON, "", "  ")
			if err != nil {
				g.logger.Warning(fmt.Sprintf("序列化转换后的决策失败: %v", err))
				return g.makeSimpleDecision(), nil
			}

			return string(convertedBytes), nil
		}
	}

	// Try to parse as TradingPairsResponse (new format with trading_pairs structure)
	// 尝试解析为 TradingPairsResponse（新格式，包含 trading_pairs 结构）
	var tradingPairsResp TradingPairsResponse
	if err := sonic.Unmarshal([]byte(trimmed), &tradingPairsResp); err == nil && len(tradingPairsResp.TradingPairs) > 0 {
		g.logger.Success(fmt.Sprintf("✅ 成功解析为 trading_pairs 格式，包含 %d 个交易对", len(tradingPairsResp.TradingPairs)))

		// Convert TradingPairsResponse to legacy format for backward compatibility
		// 将 TradingPairsResponse 转换为旧格式以保持向后兼容
		convertedJSON, err := convertTradingPairsToLegacyFormat(tradingPairsResp.TradingPairs, g.logger)
		if err != nil {
			g.logger.Warning(fmt.Sprintf("转换 trading_pairs 格式失败: %v", err))
			return g.makeSimpleDecision(), nil
		}

		// Log converted decisions
		// 记录转换后的决策
		for symbol, decision := range convertedJSON {
			g.logger.Info(fmt.Sprintf("  - %s: Action=%s, Confidence=%.2f, Leverage=%d",
				symbol, decision.Action, decision.Confidence, decision.Leverage))
		}

		// Convert back to JSON string for return
		// 转换回 JSON 字符串以返回
		convertedBytes, err := sonic.MarshalIndent(convertedJSON, "", "  ")
		if err != nil {
			g.logger.Warning(fmt.Sprintf("序列化转换后的决策失败: %v", err))
			return g.makeSimpleDecision(), nil
		}

		return string(convertedBytes), nil
	}

	// Fallback: Try multi-symbol format: map[string]TradeDecision
	// 回退：尝试多币种格式：map[string]TradeDecision
	var multi map[string]TradeDecision
	if err := sonic.Unmarshal([]byte(trimmed), &multi); err == nil && len(multi) > 0 {
		g.logger.Success(fmt.Sprintf("✅ 成功解析为多交易对格式（旧格式），包含 %d 个交易对", len(multi)))
		for sym, d := range multi {
			g.logger.Info(fmt.Sprintf("  - %s: Action=%s, Confidence=%.2f, Leverage=%d",
				sym, d.Action, d.Confidence, d.Leverage))
			// Validate at least one decision
			// 至少验证一个决策
			if strings.TrimSpace(d.Action) == "" || strings.TrimSpace(d.Symbol) == "" {
				if d.Symbol == "" {
					d.Symbol = sym
				}
			}
		}
		return response.Content, nil
	}

	// Fallback: single-object format
	// 回退到单对象格式
	g.logger.Info("尝试解析为单交易对格式...")
	var single TradeDecision
	if err := sonic.Unmarshal([]byte(trimmed), &single); err == nil && single.Symbol != "" {
		g.logger.Success("✅ 成功解析为单交易对格式")
		g.logger.Info(fmt.Sprintf("  - %s: Action=%s, Confidence=%.2f, Leverage=%d",
			single.Symbol, single.Action, single.Confidence, single.Leverage))
		return response.Content, nil
	}

	// All parsing attempts failed
	// 所有解析尝试都失败
	g.logger.Warning(fmt.Sprintf("❌ JSON 解析失败，原始响应: %s", response.Content))
	g.logger.Warning("降级到简单规则决策")
	return g.makeSimpleDecision(), nil
}

// Run executes the trading graph
func (g *SimpleTradingGraph) Run(ctx context.Context) (map[string]any, error) {
	g.logger.Header("启动交易分析工作流", '=', 80)

	compiled, err := g.BuildGraph(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to build graph: %w", err)
	}

	input := map[string]any{
		"timeframe": g.config.CryptoTimeframe,
	}

	result, err := compiled.Invoke(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("graph execution failed: %w", err)
	}

	g.logger.Header("工作流执行完成", '=', 80)

	return result, nil
}

// GetState returns the current agent state
func (g *SimpleTradingGraph) GetState() *AgentState {
	return g.state
}

// extractJSONPayload tries to extract pure JSON content from Markdown or verbose responses
// extractJSONPayload 尝试从 Markdown 或含额外内容的响应中提取纯 JSON 内容
func extractJSONPayload(content string) string {
	trimmed := strings.TrimSpace(content)

	if strings.HasPrefix(trimmed, "```") {
		// Regex captures the JSON block inside ```json ... ``` fences
		// 正则用于捕获 ```json ... ``` 中的 JSON 内容
		re := regexp.MustCompile("(?s)```(?:json)?\\s*(\\{.*\\})\\s*```")
		if matches := re.FindStringSubmatch(trimmed); len(matches) > 1 {
			return matches[1]
		}
	}

	return content
}

// formatLargeNumber formats large numbers into readable format (B/M/K)
// formatLargeNumber 将大数字格式化为易读格式（B/M/K）
func formatLargeNumber(value float64) string {
	absValue := value
	if absValue < 0 {
		absValue = -absValue
	}

	var formatted string
	if absValue >= 1e9 {
		// Billions / 十亿
		formatted = fmt.Sprintf("$%.3fB", value/1e9)
	} else if absValue >= 1e6 {
		// Millions / 百万
		formatted = fmt.Sprintf("$%.3fM", value/1e6)
	} else if absValue >= 1e3 {
		// Thousands / 千
		formatted = fmt.Sprintf("$%.3fK", value/1e3)
	} else {
		// Less than 1000 / 小于1000
		formatted = fmt.Sprintf("$%.3f", value)
	}

	return formatted
}

// convertTradingPairsToLegacyFormat converts TradingPairsResponse to legacy map[string]TradeDecision format
// convertTradingPairsToLegacyFormat 将 TradingPairsResponse 转换为旧的 map[string]TradeDecision 格式
func convertTradingPairsToLegacyFormat(tradingPairs map[string]*SymbolDecision, log *logger.ColorLogger) (map[string]TradeDecision, error) {
	result := make(map[string]TradeDecision)

	for symbol, decision := range tradingPairs {
		if decision == nil {
			log.Warning(fmt.Sprintf("交易对 %s 的决策为空，跳过", symbol))
			continue
		}

		// 🔍 打印原始决策结构的关键字段
		// Print key fields from the raw decision structure
		log.Header(fmt.Sprintf("🔍 解析 %s 决策结构", symbol), '-', 60)

		// Print decision_gate
		if decision.DecisionGate != nil {
			log.Info(fmt.Sprintf("decision_gate.value: %s", decision.DecisionGate.Value))
			if decision.DecisionGate.ReasonCode != nil {
				log.Info(fmt.Sprintf("decision_gate.reason_code: %v", decision.DecisionGate.ReasonCode.Value))
			}
		} else {
			log.Warning("decision_gate: nil")
		}

		// Print trading_signal
		if decision.TradingSignal != nil {
			if decision.TradingSignal.Action != nil {
				log.Info(fmt.Sprintf("trading_signal.action.value: %s", decision.TradingSignal.Action.Value))
			} else {
				log.Warning("trading_signal.action: nil")
			}
			log.Info(fmt.Sprintf("trading_signal.stop_loss: %.2f", decision.TradingSignal.StopLoss))
			log.Info(fmt.Sprintf("trading_signal.position_size: %.2f", decision.TradingSignal.PositionSize))
		} else {
			log.Warning("trading_signal: nil")
		}

		// Print confidence_to_leverage
		if decision.ConfidenceToLeverage != nil {
			if decision.ConfidenceToLeverage.Confidence != nil {
				log.Info(fmt.Sprintf("confidence_to_leverage.confidence.value: %.2f", decision.ConfidenceToLeverage.Confidence.Value))
			} else {
				log.Warning("confidence_to_leverage.confidence: nil")
			}
			if decision.ConfidenceToLeverage.Leverage != nil {
				log.Info(fmt.Sprintf("confidence_to_leverage.leverage.value: %d (min: %d, max: %d)",
					decision.ConfidenceToLeverage.Leverage.Value,
					decision.ConfidenceToLeverage.Leverage.Min,
					decision.ConfidenceToLeverage.Leverage.Max))
			} else {
				log.Warning("confidence_to_leverage.leverage: nil")
			}
		} else {
			log.Warning("confidence_to_leverage: nil")
		}

		// Extract action from trading_signal, but respect decision_gate
		// 从 trading_signal 提取动作，但要尊重 decision_gate
		//
		// 执行交易需要同时满足两个条件：
		// 1. decision_gate.value = "TRADE" (决策门允许交易)
		// 2. trading_signal.action.value = "BUY" 或 "SELL" (有明确的买卖信号)
		action := "HOLD"

		// Step 1: Check decision_gate
		// 步骤 1: 检查 decision_gate
		decisionGateAllowsTrade := false
		if decision.DecisionGate != nil {
			decisionGateValue := strings.ToUpper(strings.TrimSpace(decision.DecisionGate.Value))
			log.Info(fmt.Sprintf("  decision_gate.value = %s", decisionGateValue))

			if decisionGateValue == "TRADE" {
				decisionGateAllowsTrade = true
			} else if decisionGateValue == "NO_TRADE" {
				decisionGateAllowsTrade = false
				log.Info("  ⚠️  decision_gate = NO_TRADE，不允许交易")
			} else {
				log.Warning(fmt.Sprintf("  ⚠️  未知的 decision_gate 值: %s，默认不允许交易", decisionGateValue))
			}
		} else {
			// No decision_gate, assume trading is allowed (backward compatibility)
			// 没有 decision_gate，假设允许交易（向后兼容）
			decisionGateAllowsTrade = true
			log.Info("  ℹ️  无 decision_gate，默认允许交易")
		}

		// Step 2: Check trading_signal.action
		// 步骤 2: 检查 trading_signal.action
		tradingSignalAction := "HOLD"
		if decision.TradingSignal != nil && decision.TradingSignal.Action != nil {
			tradingSignalAction = strings.ToUpper(strings.TrimSpace(decision.TradingSignal.Action.Value))
			log.Info(fmt.Sprintf("  trading_signal.action.value = %s", tradingSignalAction))
		} else {
			log.Warning("  ⚠️  trading_signal.action 为空，默认 HOLD")
		}

		// Step 3: Determine final action based on both conditions
		// 步骤 3: 根据两个条件确定最终动作
		if decisionGateAllowsTrade {
			// decision_gate allows trading, check if trading_signal has BUY/SELL
			// decision_gate 允许交易，检查 trading_signal 是否有 BUY/SELL
			if tradingSignalAction == "BUY" || tradingSignalAction == "SELL" ||
				tradingSignalAction == "CLOSE_LONG" || tradingSignalAction == "CLOSE_SHORT" {
				action = tradingSignalAction
				log.Info(fmt.Sprintf("  ✅ 满足交易条件: decision_gate=TRADE + action=%s → 执行 %s", tradingSignalAction, action))
			} else {
				action = "HOLD"
				log.Info(fmt.Sprintf("  ⚠️  decision_gate=TRADE 但 action=%s → 观望", tradingSignalAction))
			}
		} else {
			// decision_gate does not allow trading, force HOLD
			// decision_gate 不允许交易，强制 HOLD
			action = "HOLD"
			log.Info(fmt.Sprintf("  ⚠️  decision_gate 不允许交易 → 强制 HOLD (忽略 trading_signal.action=%s)", tradingSignalAction))
		}

		// Extract confidence from confidence_to_leverage
		// 从 confidence_to_leverage 提取置信度
		confidence := 0.5
		if decision.ConfidenceToLeverage != nil && decision.ConfidenceToLeverage.Confidence != nil {
			confidence = decision.ConfidenceToLeverage.Confidence.Value
		}

		// Extract leverage from confidence_to_leverage
		// 从 confidence_to_leverage 提取杠杆
		leverage := 1
		if decision.ConfidenceToLeverage != nil && decision.ConfidenceToLeverage.Leverage != nil {
			leverage = decision.ConfidenceToLeverage.Leverage.Value
		}

		// Extract position size from trading_signal
		// 从 trading_signal 提取仓位大小
		positionSize := 0.0
		if decision.TradingSignal != nil {
			positionSize = decision.TradingSignal.PositionSize
		}

		// Extract stop loss from trading_signal or risk_metrics
		// 从 trading_signal 或 risk_metrics 提取止损
		stopLoss := 0.0
		if decision.TradingSignal != nil && decision.TradingSignal.StopLoss > 0 {
			stopLoss = decision.TradingSignal.StopLoss
		} else if decision.RiskMetrics != nil && decision.RiskMetrics.StopLoss != nil {
			stopLoss = decision.RiskMetrics.StopLoss.Value
		}

		// Check for position adjustment stop loss strategy (for HOLD actions)
		// 检查仓位调整止损策略（用于 HOLD 动作）
		var stopLossAdjustmentStrategy *string
		var stopLossReason *string
		if decision.PositionAdjustment != nil {
			if decision.PositionAdjustment.StopLossAdjustment != nil && decision.PositionAdjustment.StopLossAdjustment.Value != "" {
				strategy := decision.PositionAdjustment.StopLossAdjustment.Value
				stopLossAdjustmentStrategy = &strategy
			}
			if decision.PositionAdjustment.Reasoning != "" {
				reason := decision.PositionAdjustment.Reasoning
				stopLossReason = &reason
			}
		}

		// Extract reasoning from trading_signal
		// 从 trading_signal 提取理由
		reasoning := ""
		if decision.TradingSignal != nil {
			reasoning = decision.TradingSignal.Reasoning
		}

		// Extract risk/reward ratio from risk_metrics
		// 从 risk_metrics 提取盈亏比
		riskRewardRatio := 0.0
		if decision.RiskMetrics != nil && decision.RiskMetrics.EstimatedRiskReward != nil {
			riskRewardRatio = decision.RiskMetrics.EstimatedRiskReward.Value
		}

		// Build summary from trend analyzer and decision gate
		// 从趋势分析器和决策门构建总结
		summary := ""
		if decision.TrendAnalyzer != nil {
			trend := "UNKNOWN"
			phase := "UNKNOWN"
			risk := "UNKNOWN"
			if decision.TrendAnalyzer.Trend != nil {
				trend = decision.TrendAnalyzer.Trend.Value
			}
			if decision.TrendAnalyzer.Phase != nil {
				phase = decision.TrendAnalyzer.Phase.Value
			}
			if decision.TrendAnalyzer.Risk != nil {
				risk = decision.TrendAnalyzer.Risk.Value
			}
			summary = fmt.Sprintf("趋势:%s 阶段:%s 风险:%s", trend, phase, risk)
		}

		// Add decision gate info to summary
		// 将决策门信息添加到总结
		if decision.DecisionGate != nil {
			summary += fmt.Sprintf(" 决策:%s", decision.DecisionGate.Value)
			if decision.DecisionGate.ReasonCode != nil && len(decision.DecisionGate.ReasonCode.Value) > 0 {
				summary += fmt.Sprintf(" 原因:%v", decision.DecisionGate.ReasonCode.Value)
			}
		}

		// Create TradeDecision
		// 创建 TradeDecision
		tradeDecision := TradeDecision{
			Symbol:                     symbol,
			Action:                     action,
			Confidence:                 confidence,
			Leverage:                   leverage,
			PositionSize:               positionSize,
			StopLoss:                   stopLoss,
			Reasoning:                  reasoning,
			RiskRewardRatio:            riskRewardRatio,
			Summary:                    summary,
			StopLossAdjustmentStrategy: stopLossAdjustmentStrategy,
			StopLossReason:             stopLossReason,
		}

		result[symbol] = tradeDecision
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("没有成功转换任何交易对决策")
	}

	return result, nil
}

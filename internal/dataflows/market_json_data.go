package dataflows

import (
	"context"
	"fmt"
	"math"

	"github.com/bytedance/sonic"
	"github.com/eino-contrib/jsonschema"
)

// SchemaField represents a field schema description
// SchemaField 表示字段 schema 描述
type SchemaField struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

// MarketJSONData represents the structured market data for LLM consumption
// MarketJSONData 表示用于 LLM 消费的结构化市场数据
type MarketJSONData struct {
	Schema map[string]*SchemaField      `json:"schema"` // Schema descriptions for key fields
	Data   map[string]*SymbolMarketData `json:"data"`   // Key: symbol (e.g., "BTC/USDT")
}

// SymbolMarketData represents all market data for a single symbol
// SymbolMarketData 表示单个交易对的所有市场数据
type SymbolMarketData struct {
	CurrentPrice     float64                      `json:"current_price"`
	Timeframe        string                       `json:"timeframe"`
	Indicators       *IndicatorsData              `json:"indicators"`
	Volume           *VolumeData                  `json:"volume"`
	PriceHistory     *PriceHistoryData            `json:"price_history"`
	IndicatorHistory *IndicatorHistoryData        `json:"indicator_history"`
	MultiTF          map[string]*MultiTFIndicator `json:"multi_tf"` // Key: timeframe (e.g., "5m", "15m", "1h", "4h")
	MarketStats      *MarketStatsData             `json:"market_stats"`
	LongTerm1H       *LongTermData                `json:"long_term_1h"`
	Positions        *PositionsData               `json:"positions"`
}

// IndicatorsData represents technical indicators
// IndicatorsData 表示技术指标
type IndicatorsData struct {
	EMA  map[string]float64  `json:"EMA"` // Key: period (e.g., "12", "26")
	MACD float64             `json:"MACD"`
	RSI  map[string]float64  `json:"RSI"` // Key: period (e.g., "7", "14")
	ADX  float64             `json:"ADX"`
	BB   map[string]*BBBands `json:"BB"` // Key: timeframe (e.g., "15m")

	VWAP *VWAPData `json:"VWAP"`
}

// BBBands represents upper and lower Bollinger Bands
// BBBands 表示布林带上下轨
type BBBands struct {
	Upper []float64 `json:"upper"`
	Lower []float64 `json:"lower"`
}

// VWAPData represents VWAP and deviation data
// VWAPData 表示 VWAP 和偏离数据
type VWAPData struct {
	H24              float64          `json:"24h"`
	CurrentDeviation float64          `json:"current_deviation"`
	History15m       *VWAPHistoryData `json:"history_15m"`
}

// VWAPHistoryData represents VWAP historical deviation data
// VWAPHistoryData 表示 VWAP 历史偏离数据
type VWAPHistoryData struct {
	Price        []float64 `json:"price"`
	DeviationPct []float64 `json:"deviation_pct"`
}

// VolumeData represents volume information
// VolumeData 表示成交量信息
type VolumeData struct {
	Current   float64 `json:"current"`
	Average7  float64 `json:"average_7"`
	Average14 float64 `json:"average_14"`
	Average20 float64 `json:"average_20"`
}

// PriceHistoryData represents price history for different timeframes
// PriceHistoryData 表示不同时间框架的价格历史
type PriceHistoryData struct {
	TF15m *PriceSeriesData `json:"15m"`
}

// PriceSeriesData represents a series of prices
// PriceSeriesData 表示价格序列
type PriceSeriesData struct {
	Mid []float64 `json:"mid"` // Mid price (average of high and low)
}

// IndicatorHistoryData represents historical indicator values
// IndicatorHistoryData 表示历史指标值
type IndicatorHistoryData struct {
	TF15m *IndicatorSeriesData `json:"15m"`
}

// IndicatorSeriesData represents a series of indicator values
// IndicatorSeriesData 表示指标值序列
type IndicatorSeriesData struct {
	EMA12 []float64 `json:"EMA12"`
	MACD  []float64 `json:"MACD"`
	RSI7  []float64 `json:"RSI7"`
	RSI14 []float64 `json:"RSI14"`
	ADX   []float64 `json:"ADX"`
}

// MultiTFIndicator represents indicators for a single timeframe
// MultiTFIndicator 表示单个时间框架的指标
type MultiTFIndicator struct {
	EMA20 float64 `json:"EMA20"`
	EMA50 float64 `json:"EMA50"`
	MACD  float64 `json:"MACD"`
	RSI7  float64 `json:"RSI7"`
	RSI14 float64 `json:"RSI14"`
}

// MarketStatsData represents 24-hour market statistics
// MarketStatsData 表示 24 小时市场统计
type MarketStatsData struct {
	FundingRate       float64 `json:"funding_rate"`
	PriceChange24hPct float64 `json:"price_change_24h_pct"`
	High24h           float64 `json:"high_24h"`
	Low24h            float64 `json:"low_24h"`
}

// LongTermData represents long-term (1h) data
// LongTermData 表示长期（1小时）数据
type LongTermData struct {
	PriceMid []float64          `json:"price_mid"`
	EMA      map[string]float64 `json:"EMA"` // Key: period (e.g., "20", "50")
	ATR      map[string]float64 `json:"ATR"` // Key: period (e.g., "3", "7", "14")
	MACD     []float64          `json:"MACD"`
	RSI14    []float64          `json:"RSI14"`
	Volume   *VolumeData        `json:"volume"` // 1h 时间框架的成交量数据
}

// PositionsData represents position and open interest data
// PositionsData 表示持仓和未平仓合约数据
type PositionsData struct {
	OpenInterest *OpenInterestData `json:"open_interest"`
}

// OpenInterestData represents open interest statistics
// OpenInterestData 表示未平仓合约统计
type OpenInterestData struct {
	TF15m      *OITimeframeData `json:"15m"`
	DailyStats *DailyStatsData  `json:"daily_stats"`
}

// OITimeframeData represents open interest data for a specific timeframe
// OITimeframeData 表示特定时间框架的未平仓合约数据
type OITimeframeData struct {
	ChangeRate []float64 `json:"change_rate"` // Change rate relative to previous point (%)
	Volume     []float64 `json:"volume"`      // Open interest volume
}

// DailyStatsData represents daily statistics
// DailyStatsData 表示每日统计数据
type DailyStatsData struct {
	PriceChangePercent float64 `json:"price_change_percent"`
	High               float64 `json:"high"`
	Low                float64 `json:"low"`
	Volume             float64 `json:"volume"`
}

func round2(val float64) float64 {
	pow := math.Pow(10, float64(2))
	return math.Round(val*pow) / pow
}

func roundSlice2(data []float64) []float64 {
	out := make([]float64, len(data))
	for i, v := range data {
		if math.IsNaN(v) {
			out[i] = math.NaN()
		} else {
			out[i] = round2(v)
		}
	}
	return out
}

// BuildMarketJSONData builds structured market data for a single symbol
// BuildMarketJSONData 为单个交易对构建结构化市场数据
func BuildMarketJSONData(
	ctx context.Context,
	marketData *MarketData,
	symbol string,
	timeframe string,
	ohlcvData []OHLCV,
	indicators *TechnicalIndicators,
	longerIndicators *TechnicalIndicators,
	longerOHLCV []OHLCV,
) (*SymbolMarketData, error) {
	if len(ohlcvData) == 0 {
		return nil, fmt.Errorf("no OHLCV data available")
	}

	lastIdx := len(ohlcvData) - 1
	currentPrice := ohlcvData[lastIdx].Close

	// Build indicators data
	// 构建指标数据
	indicatorsData := buildIndicatorsData(ohlcvData, indicators, timeframe)

	// Fill VWAP history data
	// 填充 VWAP 历史数据
	vwapHistory, err := marketData.GetVWAPDeviationHistory(ctx, symbol, timeframe, 10)
	if err == nil {
		if priceHist, ok := vwapHistory["price_history"].([]float64); ok {
			if devHist, ok := vwapHistory["deviation_history"].([]float64); ok {
				indicatorsData.VWAP.History15m = &VWAPHistoryData{
					Price:        roundSlice2(priceHist),
					DeviationPct: roundSlice2(devHist),
				}
			}
		}
		if currentDev, ok := vwapHistory["current_deviation"].(float64); ok {
			indicatorsData.VWAP.CurrentDeviation = round2(currentDev)
		}
		if vwap24h, ok := vwapHistory["vwap_24h"].(float64); ok {
			indicatorsData.VWAP.H24 = round2(vwap24h)
		}
	}

	// Build volume data
	// 构建成交量数据
	volumeData := buildVolumeData(ohlcvData, timeframe)

	// Build price history
	// 构建价格历史
	priceHistory := buildPriceHistory(ohlcvData, longerOHLCV)

	// Build indicator history
	// 构建指标历史
	indicatorHistory := buildIndicatorHistory(ohlcvData, indicators)

	// Build multi-timeframe indicators
	// 构建多时间框架指标
	multiTF, err := buildMultiTimeframeIndicators(ctx, marketData, symbol)
	if err != nil {
		// Log error but continue with empty multi-TF data
		// 记录错误但继续使用空的多时间框架数据
		multiTF = make(map[string]*MultiTFIndicator)
	}

	// Build market stats
	// 构建市场统计
	marketStats, err := buildMarketStats(ctx, marketData, symbol)
	if err != nil {
		// Log error but continue with empty stats
		// 记录错误但继续使用空的统计数据
		marketStats = &MarketStatsData{}
	}

	// Build long-term data
	// 构建长期数据
	longTermData := buildLongTermData(longerOHLCV, longerIndicators)

	// Build positions data
	// 构建持仓数据
	positionsData, err := buildPositionsData(ctx, marketData, symbol)
	if err != nil {
		// Log error but continue with empty positions data
		// 记录错误但继续使用空的持仓数据
		positionsData = &PositionsData{}
	}

	return &SymbolMarketData{
		CurrentPrice:     currentPrice,
		Timeframe:        timeframe,
		Indicators:       indicatorsData,
		Volume:           volumeData,
		PriceHistory:     priceHistory,
		IndicatorHistory: indicatorHistory,
		MultiTF:          multiTF,
		MarketStats:      marketStats,
		LongTerm1H:       longTermData,
		Positions:        positionsData,
	}, nil
}

// buildIndicatorsData builds indicators data from OHLCV and technical indicators
// buildIndicatorsData 从 OHLCV 和技术指标构建指标数据
func buildIndicatorsData(ohlcvData []OHLCV, indicators *TechnicalIndicators, timeframe string) *IndicatorsData {
	// Helper function to get last valid value
	// 辅助函数：获取最后一个有效值
	getLastValue := func(data []float64) float64 {
		if len(data) == 0 {
			return 0.0
		}
		for i := len(data) - 1; i >= 0; i-- {
			if !math.IsNaN(data[i]) {
				return data[i]
			}
		}
		return 0.0
	}

	// Helper function to get last N values
	// 辅助函数：获取最后 N 个值
	getLastNValues := func(data []float64, n int) []float64 {
		if len(data) == 0 {
			return []float64{}
		}
		startIdx := len(data) - n
		if startIdx < 0 {
			startIdx = 0
		}
		result := make([]float64, 0, n)
		for i := startIdx; i < len(data); i++ {
			if !math.IsNaN(data[i]) {
				result = append(result, data[i])
			}
		}
		return result
	}

	// Build EMA map
	// 构建 EMA 映射
	emaMap := map[string]float64{
		"12": round2(getLastValue(indicators.EMA_12)),
		"26": round2(getLastValue(indicators.EMA_26)),
	}

	// Build RSI map
	// 构建 RSI 映射
	rsiMap := map[string]float64{
		"7":  round2(getLastValue(indicators.RSI_7)),
		"14": round2(getLastValue(indicators.RSI)),
	}

	// Build ATR map
	// 构建 ATR 映射,不要影响趋势
	// atrMap := map[string]float64{
	// 	"3":  round2(getLastValue(indicators.ATR_3)),
	// 	"7":  round2(getLastValue(indicators.ATR_7)),
	// 	"14": round2(getLastValue(indicators.ATR_14)),
	// }

	// Build BB data (last 10 values)
	// 构建布林带数据（最后 10 个值）
	bbData := map[string]*BBBands{
		timeframe: {
			Upper: roundSlice2(getLastNValues(indicators.BB_Upper, 10)),
			Lower: roundSlice2(getLastNValues(indicators.BB_Lower, 10)),
		},
	}

	// Build VWAP data
	// 构建 VWAP 数据
	vwapData := &VWAPData{
		H24:              round2(getLastValue(indicators.VWAP)),
		CurrentDeviation: 0.0, // Will be filled by buildVWAPHistory
		History15m:       nil, // Will be filled by buildVWAPHistory
	}

	return &IndicatorsData{
		EMA:  emaMap,
		MACD: round2(getLastValue(indicators.MACD)),
		RSI:  rsiMap,
		ADX:  round2(getLastValue(indicators.ADX)),
		BB:   bbData,

		VWAP: vwapData,
	}
}

// buildVolumeData builds volume data from OHLCV
// buildVolumeData 从 OHLCV 构建成交量数据
func buildVolumeData(ohlcvData []OHLCV, timeframe string) *VolumeData {
	if len(ohlcvData) == 0 {
		return &VolumeData{}
	}

	// GetOHLCV already excludes the last incomplete candle
	// GetOHLCV 已经排除了最后一根未完成的K线
	lastIdx := len(ohlcvData) - 1
	currentVolume := ohlcvData[lastIdx].Volume

	// Helper function to calculate average volume for N periods
	// 辅助函数：计算 N 个周期的平均成交量
	calcAvgVolume := func(periods int) float64 {
		n := periods
		if len(ohlcvData) < n {
			n = len(ohlcvData)
		}
		sum := 0.0
		for i := lastIdx - n + 1; i <= lastIdx; i++ {
			if i >= 0 {
				sum += ohlcvData[i].Volume
			}
		}
		if n == 0 {
			return 0.0
		}
		return sum / float64(n)
	}

	// Calculate average volumes for 7, 14, and 20 periods
	// 计算 7、14 和 20 周期的平均成交量
	avg7 := calcAvgVolume(7)
	avg14 := calcAvgVolume(14)
	avg20 := calcAvgVolume(20)

	return &VolumeData{
		Current:   round2(currentVolume),
		Average7:  round2(avg7),
		Average14: round2(avg14),
		Average20: round2(avg20),
	}
}

// buildPriceHistory builds price history for different timeframes
// buildPriceHistory 构建不同时间框架的价格历史
func buildPriceHistory(ohlcvData []OHLCV, longerOHLCV []OHLCV) *PriceHistoryData {
	// Helper function to get last N mid prices
	// 辅助函数：获取最后 N 个中间价
	getMidPrices := func(data []OHLCV, n int) []float64 {
		if len(data) == 0 {
			return []float64{}
		}
		startIdx := len(data) - n
		if startIdx < 0 {
			startIdx = 0
		}
		result := make([]float64, 0, n)
		for i := startIdx; i < len(data); i++ {
			midPrice := (data[i].High + data[i].Low) / 2
			result = append(result, midPrice)
		}
		return result
	}

	return &PriceHistoryData{
		TF15m: &PriceSeriesData{
			Mid: roundSlice2(getMidPrices(ohlcvData, 10)),
		},
	}
}

// buildIndicatorHistory builds indicator history data
// buildIndicatorHistory 构建指标历史数据
func buildIndicatorHistory(ohlcvData []OHLCV, indicators *TechnicalIndicators) *IndicatorHistoryData {
	// Helper function to get last N values
	// 辅助函数：获取最后 N 个值
	getLastNValues := func(data []float64, n int) []float64 {
		if len(data) == 0 {
			return []float64{}
		}
		startIdx := len(data) - n
		if startIdx < 0 {
			startIdx = 0
		}
		result := make([]float64, 0, n)
		for i := startIdx; i < len(data); i++ {
			if !math.IsNaN(data[i]) {
				result = append(result, data[i])
			} else {
				result = append(result, 0.0)
			}
		}
		return result
	}

	return &IndicatorHistoryData{
		TF15m: &IndicatorSeriesData{
			EMA12: roundSlice2(getLastNValues(indicators.EMA_12, 10)),
			MACD:  roundSlice2(getLastNValues(indicators.MACD, 10)),
			RSI7:  roundSlice2(getLastNValues(indicators.RSI_7, 10)),
			RSI14: roundSlice2(getLastNValues(indicators.RSI, 10)),
			ADX:   roundSlice2(getLastNValues(indicators.ADX, 10)),
		},
	}
}

// buildMultiTimeframeIndicators builds multi-timeframe indicators
// buildMultiTimeframeIndicators 构建多时间框架指标
func buildMultiTimeframeIndicators(ctx context.Context, marketData *MarketData, symbol string) (map[string]*MultiTFIndicator, error) {
	result := make(map[string]*MultiTFIndicator)

	// Get multi-timeframe indicators from MarketData
	// 从 MarketData 获取多时间框架指标
	indicators := marketData.GetMultiTimeframeIndicators(ctx, symbol)

	if len(indicators) == 0 {
		return nil, fmt.Errorf("no multi-timeframe indicators available")
	}

	for _, ind := range indicators {
		result[ind.Timeframe] = &MultiTFIndicator{
			EMA20: round2(ind.EMA20),
			EMA50: round2(ind.EMA50),
			MACD:  round2(ind.MACD),
			RSI7:  round2(ind.RSI7),
			RSI14: round2(ind.RSI14),
		}
	}

	return result, nil
}

// buildMarketStats builds market statistics data
// buildMarketStats 构建市场统计数据
func buildMarketStats(ctx context.Context, marketData *MarketData, symbol string) (*MarketStatsData, error) {
	// Get funding rate
	// 获取资金费率
	fundingRate, err := marketData.GetFundingRate(ctx, symbol)
	if err != nil {
		fundingRate = 0.0
	}

	// Get 24-hour stats
	// 获取 24 小时统计
	stats, err := marketData.Get24HrStats(ctx, symbol)
	if err != nil {
		return &MarketStatsData{
			FundingRate: fundingRate,
		}, nil
	}

	// Parse stats
	// 解析统计数据
	priceChangePct := 0.0
	high24h := 0.0
	low24h := 0.0
	volume := 0.0

	if val, ok := stats["price_change_percent"]; ok {
		fmt.Sscanf(val, "%f", &priceChangePct)
	}
	if val, ok := stats["high_price"]; ok {
		fmt.Sscanf(val, "%f", &high24h)
	}
	if val, ok := stats["low_price"]; ok {
		fmt.Sscanf(val, "%f", &low24h)
	}
	if val, ok := stats["volume"]; ok {
		fmt.Sscanf(val, "%f", &volume)
	}

	return &MarketStatsData{
		FundingRate:       fundingRate,
		PriceChange24hPct: priceChangePct,
		High24h:           high24h,
		Low24h:            low24h,
	}, nil
}

// buildLongTermData builds long-term (1h) data
// buildLongTermData 构建长期（1小时）数据
func buildLongTermData(longerOHLCV []OHLCV, longerIndicators *TechnicalIndicators) *LongTermData {
	if len(longerOHLCV) == 0 || longerIndicators == nil {
		return &LongTermData{}
	}

	// Helper function to get last valid value
	// 辅助函数：获取最后一个有效值
	getLastValue := func(data []float64) float64 {
		if len(data) == 0 {
			return 0.0
		}
		for i := len(data) - 1; i >= 0; i-- {
			if !math.IsNaN(data[i]) {
				return data[i]
			}
		}
		return 0.0
	}

	// Helper function to get last N values
	// 辅助函数：获取最后 N 个值
	getLastNValues := func(data []float64, n int) []float64 {
		if len(data) == 0 {
			return []float64{}
		}
		startIdx := len(data) - n
		if startIdx < 0 {
			startIdx = 0
		}
		result := make([]float64, 0, n)
		for i := startIdx; i < len(data); i++ {
			if !math.IsNaN(data[i]) {
				result = append(result, data[i])
			}
		}
		return result
	}

	// Get last N mid prices
	// 获取最后 N 个中间价
	n := 10
	startIdx := len(longerOHLCV) - n
	if startIdx < 0 {
		startIdx = 0
	}
	priceMid := make([]float64, 0, n)
	for i := startIdx; i < len(longerOHLCV); i++ {
		midPrice := (longerOHLCV[i].High + longerOHLCV[i].Low) / 2
		priceMid = append(priceMid, midPrice)
	}

	// Build volume data for 1h timeframe
	// 构建 1h 时间框架的成交量数据
	volumeData := buildVolumeData(longerOHLCV, "1h")

	return &LongTermData{
		PriceMid: roundSlice2(priceMid),
		EMA: map[string]float64{
			"20": round2(getLastValue(longerIndicators.EMA_20)),
			"50": round2(getLastValue(longerIndicators.EMA_50)),
		},
		ATR: map[string]float64{
			// "3":  round2(getLastValue(longerIndicators.ATR_3)),
			"7": round2(getLastValue(longerIndicators.ATR_7)),
			// "14": round2(getLastValue(longerIndicators.ATR_14)),
		},
		MACD:   roundSlice2(getLastNValues(longerIndicators.MACD, 10)),
		RSI14:  roundSlice2(getLastNValues(longerIndicators.RSI, 10)),
		Volume: volumeData,
	}
}

// buildPositionsData builds positions and open interest data
// buildPositionsData 构建持仓和未平仓合约数据
func buildPositionsData(ctx context.Context, marketData *MarketData, symbol string) (*PositionsData, error) {
	// Get open interest change data for 15m timeframe (4 hours = 16 periods)
	// 获取 15 分钟时间框架的持仓量变化数据（4 小时 = 16 个周期）
	oiData, err := marketData.GetOpenInterestChange(ctx, symbol, "15m", 16)
	if err != nil {
		return nil, err
	}

	// Extract change rates and volumes
	// 提取变化率和持仓量
	changeRates := []float64{}
	volumes := []float64{}

	if rates, ok := oiData["change_rates"].([]float64); ok {
		changeRates = rates
	}
	if vols, ok := oiData["series_volumes"].([]float64); ok {
		volumes = vols
	}

	// Get 24-hour stats for daily statistics
	// 获取 24 小时统计数据
	stats, err := marketData.Get24HrStats(ctx, symbol)
	if err != nil {
		stats = make(map[string]string)
	}

	// Parse daily stats
	// 解析每日统计
	priceChangePct := 0.0
	high24h := 0.0
	low24h := 0.0
	volume24h := 0.0

	if val, ok := stats["price_change_percent"]; ok {
		fmt.Sscanf(val, "%f", &priceChangePct)
	}
	if val, ok := stats["high_price"]; ok {
		fmt.Sscanf(val, "%f", &high24h)
	}
	if val, ok := stats["low_price"]; ok {
		fmt.Sscanf(val, "%f", &low24h)
	}
	if val, ok := stats["volume"]; ok {
		fmt.Sscanf(val, "%f", &volume24h)
	}

	return &PositionsData{
		OpenInterest: &OpenInterestData{
			TF15m: &OITimeframeData{
				ChangeRate: roundSlice2(changeRates),
				Volume:     volumes,
			},
			DailyStats: &DailyStatsData{
				PriceChangePercent: priceChangePct,
				High:               high24h,
				Low:                low24h,
				Volume:             volume24h,
			},
		},
	}, nil
}

// GetDefaultSchema returns the default schema field descriptions
// GetDefaultSchema 返回默认的 schema 字段描述
func GetDefaultSchema() map[string]*SchemaField {
	return map[string]*SchemaField{
		"VWAP": {
			Type:        "float",
			Description: "24H滚动成交量加权平均价格，可作为日内支撑/阻力参考",
		},
		"multi_tf": {
			Type:        "object",
			Description: "多时间框架指标参考，用于趋势确认和执行决策，包含 EMA、MACD、RSI 等",
		},
		"long_term_1h": {
			Type:        "object",
			Description: "1小时以内的长期数据，用于趋势和波动分析，包括价格序列、EMA、ATR、MACD、RSI",
		},
		"price_history": {
			Type:        "object",
			Description: "key 为时间周期（如 15m, 1h），值为中间价序列，每个点为 (bid+ask)/2",
		},
		"BB": {
			Type:        "object",
			Description: "布林带上下轨，用于参考支撑/阻力，key 为 upper/lower",
		},
		"current_position": {
			Type:        "object|null",
			Description: "仅用于仓位管理判断，不得用于趋势或加仓决策",
		},
	}
}

// GenerateMarketDataSchema generates JSON Schema for MarketJSONData
// GenerateMarketDataSchema 为 MarketJSONData 生成 JSON Schema
func GenerateMarketDataSchema() (string, error) {
	// Use jsonschema to reflect the structure
	// 使用 jsonschema 反射结构
	schema := jsonschema.Reflect(&MarketJSONData{})

	// Marshal to JSON string
	// 序列化为 JSON 字符串
	schemaBytes, err := sonic.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal schema: %w", err)
	}

	return string(schemaBytes), nil
}

// GetMarketDataSchemaWithDescription returns a formatted schema description for LLM
// GetMarketDataSchemaWithDescription 返回格式化的 Schema 描述供 LLM 使用
func GetMarketDataSchemaWithDescription() string {
	schema, err := GenerateMarketDataSchema()
	if err != nil {
		return "# 市场数据结构\n\n无法生成 Schema"
	}

	description := `# 市场数据 JSON Schema

以下是市场数据的完整 JSON Schema 定义，描述了所有字段的类型和结构：

` + "```json\n" + schema + "\n```" + `

## 主要字段说明

### data (object)
包含所有交易对的市场数据，key 为交易对名称（如 "BTC/USDT"）

### SymbolMarketData (每个交易对的数据)

#### current_price (number)
当前价格（USDT）

#### timeframe (string)
主时间周期（如 "15m", "1h"）

#### indicators (object)
技术指标数据：
- **EMA**: 指数移动平均线，包含多个周期（12, 26, 50, 100, 200）
- **MACD**: MACD 指标当前值
- **RSI**: 相对强弱指标，包含多个周期（7, 14, 21）
- **ADX**: 平均趋向指标，衡量趋势强度
- **BB**: 布林带，包含上轨和下轨的历史数据
- **ATR**: 平均真实波幅，包含多个周期（3, 7, 14）
- **VWAP**: 成交量加权平均价格
  - 24h: 24小时 VWAP
  - current_deviation: 当前价格相对 VWAP 的偏离百分比
  - history_15m: 15分钟间隔的价格和偏离历史

#### volume (object)
成交量数据：
- current: 当前周期成交量
- average: 平均成交量
- period: 统计周期

#### price_history (object)
价格历史数据：
- 15m: 15分钟时间框架的最近10个价格点（中间价）
- 1h: 1小时时间框架的最近10个价格点（中间价）

#### indicator_history (object)
指标历史序列（最近10个数据点）：
- 15m: 包含 EMA12, MACD, RSI7, RSI14, ADX 的历史数组

#### multi_tf (object)
多时间框架指标快照：
- 包含 5m, 15m, 30m, 1h, 4h 等时间框架
- 每个时间框架包含: EMA20, EMA50, MACD, RSI7, RSI14

#### market_stats (object)
市场统计数据：
- funding_rate: 资金费率
- price_change_24h_pct: 24小时价格变化百分比
- high_24h: 24小时最高价
- low_24h: 24小时最低价

#### long_term_1h (object)
长期（1小时）时间框架数据：
- price_mid: 价格历史（最近10个数据点）
- EMA: 长期 EMA（20, 50）
- ATR: 长期 ATR（3, 7, 14）
- MACD: MACD 历史数组
- RSI14: RSI14 历史数组

#### positions (object)
持仓量数据：
- open_interest: 持仓量信息
  - 15m: 15分钟间隔的变化率和持仓量序列
  - daily_stats: 每日统计数据

## 数据使用建议

1. **趋势判断**: 结合 EMA, MACD, ADX 判断趋势方向和强度
2. **超买超卖**: 使用 RSI 判断是否超买或超卖
3. **波动性**: 使用 ATR 和 BB 评估市场波动性
4. **成交量确认**: 使用 volume 和 VWAP 确认价格走势
5. **多时间框架**: 结合 multi_tf 进行多周期分析
6. **持仓量**: 使用 positions 数据判断市场情绪
`

	return description
}

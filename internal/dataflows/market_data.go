package dataflows

import (
	"context"
	"crypto/tls"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/adshao/go-binance/v2/futures"
	"github.com/oak/crypto-trading-bot/internal/config"
)

// OHLCV represents a candlestick data point
type OHLCV struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// TechnicalIndicators holds calculated technical indicators
type TechnicalIndicators struct {
	RSI       []float64 // RSI(14) - 14期相对强弱指数
	RSI_7     []float64 // RSI(7) - 7期相对强弱指数（短期超买超卖）
	MACD      []float64
	Signal    []float64
	BB_Upper  []float64
	BB_Middle []float64
	BB_Lower  []float64
	SMA_20    []float64
	SMA_50    []float64
	SMA_200   []float64
	EMA_12    []float64
	EMA_20    []float64 // EMA(20) - 20期指数移动平均（常用趋势线）
	EMA_26    []float64
	EMA_50    []float64 // EMA(50) - 50期指数移动平均（中期趋势线）
	ATR_14    []float64 // ATR(14) - 14期平均真实波幅
	ATR_7     []float64 // ATR(7) - 7期平均真实波幅
	ATR_3     []float64 // ATR(3) - 3期平均真实波幅
	Volume    []float64
	VWAP      []float64 // VWAP - Volume Weighted Average Price 成交量加权平均价

	// New indicators for trend strength and confirmation
	// 新增指标：趋势强度和确认
	ADX         []float64 // Average Directional Index - 趋势强度
	DI_Plus     []float64 // +DI - 上升趋向指标
	DI_Minus    []float64 // -DI - 下降趋向指标
	VolumeRatio []float64 // Volume Ratio - 成交量比率
}

// MultiTimeframeIndicator holds key indicators for a single timeframe
// MultiTimeframeIndicator 存储单个时间框架的关键指标
type MultiTimeframeIndicator struct {
	Timeframe string  // 时间框架（如 "3m", "5m", "15m", "1h", "4h"）
	EMA20     float64 // EMA(20) - 20期指数移动平均
	EMA50     float64 // EMA(50) - 50期指数移动平均
	MACD      float64 // MACD - 动量指标
	RSI7      float64 // RSI(7) - 7期相对强弱指数
	RSI14     float64 // RSI(14) - 14期相对强弱指数
}

// MarketData handles crypto market data fetching
type MarketData struct {
	client *futures.Client
	config *config.Config
}

// NewMarketData creates a new MarketData instance
// Note: For public endpoints (klines, orderbook, etc.), API key is not required
func NewMarketData(cfg *config.Config) *MarketData {
	futures.UseTestnet = cfg.BinanceTestMode

	// For public data endpoints, we can use empty API credentials
	// Only private endpoints (account info, trading) require valid credentials
	apiKey := ""
	apiSecret := ""

	// If API credentials are provided, use them (for authenticated endpoints)
	if cfg.BinanceAPIKey != "" && cfg.BinanceAPISecret != "" {
		apiKey = cfg.BinanceAPIKey
		apiSecret = cfg.BinanceAPISecret
	}

	client := futures.NewClient(apiKey, apiSecret)

	// Set proxy if configured
	if cfg.BinanceProxy != "" {
		proxyURL, err := url.Parse(cfg.BinanceProxy)
		if err == nil {
			// Create custom HTTP client with proxy
			httpClient := &http.Client{
				Transport: &http.Transport{
					Proxy: http.ProxyURL(proxyURL),
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: false,
					},
				},
				Timeout: 30 * time.Second,
			}
			client.HTTPClient = httpClient
		}
	}

	return &MarketData{
		client: client,
		config: cfg,
	}
}

// GetOHLCV fetches OHLCV data for a symbol
// Note: ALWAYS excludes the last candle to ensure only completed candles are returned
// 注意：始终排除最后一根K线，确保只返回已完成的K线
// This guarantees 100% data certainty for all technical indicator calculations
// 这保证了所有技术指标计算的100%数据确定性
func (m *MarketData) GetOHLCV(ctx context.Context, symbol string, timeframe string, lookbackDays int) ([]OHLCV, error) {
	interval := convertTimeframe(timeframe)

	startTime := time.Now().AddDate(0, 0, -lookbackDays)
	endTime := time.Now()

	klines, err := m.client.NewKlinesService().
		Symbol(symbol).
		Interval(interval).
		StartTime(startTime.UnixMilli()).
		EndTime(endTime.UnixMilli()).
		Limit(1000).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch klines: %w", err)
	}

	// Always remove the last candle to ensure we only use completed candles
	// 始终移除最后一根K线，确保只使用已完成的K线
	// This guarantees data stability and eliminates any ambiguity about candle completion status
	// 这保证了数据稳定性，消除了关于K线是否完成的任何模糊性
	if len(klines) > 0 {
		klines = klines[:len(klines)-1]
	}

	ohlcvData := make([]OHLCV, 0, len(klines))
	for _, k := range klines {
		open, _ := strconv.ParseFloat(k.Open, 64)
		high, _ := strconv.ParseFloat(k.High, 64)
		low, _ := strconv.ParseFloat(k.Low, 64)
		closePrice, _ := strconv.ParseFloat(k.Close, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)

		ohlcvData = append(ohlcvData, OHLCV{
			Timestamp: time.Unix(k.OpenTime/1000, 0),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
		})
	}

	return ohlcvData, nil
}

// CalculateIndicators calculates technical indicators from OHLCV data
// Optional parameter: atrPeriod (for trailing stop ATR calculation from longer timeframe)
// 可选参数：atrPeriod（用于从长期时间周期计算追踪止损的 ATR）
func CalculateIndicators(ohlcvData []OHLCV, atrPeriod ...int) *TechnicalIndicators {
	if len(ohlcvData) == 0 {
		return &TechnicalIndicators{}
	}

	// Extract price and volume arrays
	closes := make([]float64, len(ohlcvData))
	highs := make([]float64, len(ohlcvData))
	lows := make([]float64, len(ohlcvData))
	volumes := make([]float64, len(ohlcvData))

	for i, candle := range ohlcvData {
		closes[i] = candle.Close
		highs[i] = candle.High
		lows[i] = candle.Low
		volumes[i] = candle.Volume
	}

	// Determine ATR period for trailing stop (default 14)
	// 确定追踪止损的 ATR 周期（默认 14）
	//atrPeriodValue := 7
	//if len(atrPeriod) > 0 && atrPeriod[0] > 0 {
	//	atrPeriodValue = atrPeriod[0]
	//}

	// Calculate indicators
	rsi := calculateRSI(closes, 14)
	rsi7 := calculateRSI(closes, 7) // 新增：7期RSI（短期超买超卖判断）
	macd, signal := calculateMACD(closes)
	bbUpper, bbMiddle, bbLower := calculateBollingerBands(closes, 20, 2.0)
	sma20 := calculateSMA(closes, 20)
	sma50 := calculateSMA(closes, 50)
	sma200 := calculateSMA(closes, 200)
	ema12 := calculateEMA(closes, 12)
	ema20 := calculateEMA(closes, 20) // 新增：20期EMA（常用趋势线）
	ema26 := calculateEMA(closes, 26)
	ema50 := calculateEMA(closes, 50) // 新增：50期EMA（中期趋势线）
	atr14 := calculateATR(highs, lows, closes, 14)
	atr7 := calculateATR(highs, lows, closes, 7)
	atr3 := calculateATR(highs, lows, closes, 3) // 追踪止损 ATR（周期可配置）/ Trailing stop ATR (configurable period)

	// New indicators for trend strength and volume confirmation
	// 新增指标：趋势强度和成交量确认
	adx, diPlus, diMinus := calculateADX(highs, lows, closes, 14)

	// Calculate 24h rolling VWAP window based on candle interval
	// 根据 K 线时间间隔计算 24 小时对应的窗口大小（K 线数量）
	vwapWindow := calculate24hWindowBars(ohlcvData)
	vwap := calculateVWAP(highs, lows, closes, volumes, vwapWindow)
	volumeRatio := calculateVolumeRatio(volumes, 20)

	return &TechnicalIndicators{
		RSI:       rsi,
		RSI_7:     rsi7, // 新增
		MACD:      macd,
		Signal:    signal,
		BB_Upper:  bbUpper,
		BB_Middle: bbMiddle,
		BB_Lower:  bbLower,
		SMA_20:    sma20,
		SMA_50:    sma50,
		SMA_200:   sma200,
		EMA_12:    ema12,
		EMA_20:    ema20, // 新增
		EMA_26:    ema26,
		EMA_50:    ema50, // 新增
		ATR_14:    atr14,
		ATR_7:     atr7,
		ATR_3:     atr3, // 新增
		Volume:    volumes,
		VWAP:      vwap,

		// New indicators
		// 新增指标
		ADX:         adx,
		DI_Plus:     diPlus,
		DI_Minus:    diMinus,
		VolumeRatio: volumeRatio,
	}
}

// calculate24hWindowBars infers how many candles roughly represent 24 hours
// 根据 OHLCV 时间戳推断 24 小时内包含的 K 线数量（用于 VWAP 滚动窗口）
func calculate24hWindowBars(ohlcvData []OHLCV) int {
	n := len(ohlcvData)
	if n <= 1 {
		if n == 1 {
			return 1
		}
		return 0
	}

	// Use the interval between the first two candles as the base timeframe
	// 使用前两根 K 线的时间差作为基础时间周期
	interval := ohlcvData[1].Timestamp.Sub(ohlcvData[0].Timestamp)
	if interval <= 0 {
		return n
	}

	minutes := int(interval.Minutes())
	if minutes <= 0 {
		minutes = 1
	}

	// 24 hours = 1440 minutes
	// 24 小时 = 1440 分钟
	windowBars := 1440 / minutes
	if windowBars < 1 {
		windowBars = 1
	}
	if windowBars > n {
		windowBars = n
	}

	return windowBars
}

// calculateSMA calculates Simple Moving Average
func calculateSMA(data []float64, period int) []float64 {
	result := make([]float64, len(data))
	for i := range data {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}
		sum := 0.0
		for j := 0; j < period; j++ {
			sum += data[i-j]
		}
		result[i] = sum / float64(period)
	}
	return result
}

// calculateEMA calculates Exponential Moving Average
// calculateEMA 计算指数移动平均（跳过NaN值）
func calculateEMA(data []float64, period int) []float64 {
	result := make([]float64, len(data))
	multiplier := 2.0 / float64(period+1)

	// Find first valid index (skip leading NaN values)
	// 找到第一个有效索引（跳过开头的NaN值）
	firstValidIdx := 0
	for i := 0; i < len(data); i++ {
		if !math.IsNaN(data[i]) {
			firstValidIdx = i
			break
		}
	}

	// Mark all values before we have enough data as NaN
	// 标记数据不足时的值为NaN
	for i := 0; i < firstValidIdx+period-1 && i < len(data); i++ {
		result[i] = math.NaN()
	}

	// Check if we have enough data
	// 检查是否有足够的数据
	if firstValidIdx+period > len(data) {
		return result
	}

	// Calculate first EMA value as SMA (skip NaN values)
	// 计算第一个EMA值作为SMA（跳过NaN值）
	sum := 0.0
	validCount := 0
	for i := firstValidIdx; i < firstValidIdx+period && i < len(data); i++ {
		if !math.IsNaN(data[i]) {
			sum += data[i]
			validCount++
		}
	}

	if validCount >= period {
		result[firstValidIdx+period-1] = sum / float64(validCount)
	} else {
		return result
	}

	// Calculate EMA for remaining values (skip NaN in input)
	// 计算剩余值的EMA（跳过输入中的NaN）
	for i := firstValidIdx + period; i < len(data); i++ {
		if !math.IsNaN(data[i]) && !math.IsNaN(result[i-1]) {
			result[i] = (data[i]-result[i-1])*multiplier + result[i-1]
		} else {
			result[i] = math.NaN()
		}
	}

	return result
}

// calculateRSI calculates Relative Strength Index
func calculateRSI(data []float64, period int) []float64 {
	result := make([]float64, len(data))

	if len(data) < period+1 {
		for i := range result {
			result[i] = math.NaN()
		}
		return result
	}

	gains := make([]float64, len(data))
	losses := make([]float64, len(data))

	for i := 1; i < len(data); i++ {
		change := data[i] - data[i-1]
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = -change
		}
	}

	avgGain := 0.0
	avgLoss := 0.0
	for i := 1; i <= period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	for i := 0; i < period; i++ {
		result[i] = math.NaN()
	}

	for i := period; i < len(data); i++ {
		if i == period {
			if avgLoss == 0 {
				result[i] = 100
			} else {
				rs := avgGain / avgLoss
				result[i] = 100 - (100 / (1 + rs))
			}
		} else {
			avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)

			if avgLoss == 0 {
				result[i] = 100
			} else {
				rs := avgGain / avgLoss
				result[i] = 100 - (100 / (1 + rs))
			}
		}
	}

	return result
}

// calculateMACD calculates MACD and Signal line
func calculateMACD(data []float64) ([]float64, []float64) {
	ema12 := calculateEMA(data, 12)
	ema26 := calculateEMA(data, 26)

	macd := make([]float64, len(data))
	for i := range data {
		if math.IsNaN(ema12[i]) || math.IsNaN(ema26[i]) {
			macd[i] = math.NaN()
		} else {
			macd[i] = ema12[i] - ema26[i]
		}
	}

	signal := calculateEMA(macd, 9)
	return macd, signal
}

// calculateBollingerBands calculates Bollinger Bands
func calculateBollingerBands(data []float64, period int, stdDev float64) ([]float64, []float64, []float64) {
	middle := calculateSMA(data, period)
	upper := make([]float64, len(data))
	lower := make([]float64, len(data))

	for i := range data {
		if math.IsNaN(middle[i]) {
			upper[i] = math.NaN()
			lower[i] = math.NaN()
			continue
		}

		// Calculate standard deviation
		sum := 0.0
		for j := 0; j < period; j++ {
			diff := data[i-j] - middle[i]
			sum += diff * diff
		}
		sd := math.Sqrt(sum / float64(period))

		upper[i] = middle[i] + stdDev*sd
		lower[i] = middle[i] - stdDev*sd
	}

	return upper, middle, lower
}

// calculateATR calculates Average True Range
func calculateATR(highs, lows, closes []float64, period int) []float64 {
	result := make([]float64, len(closes))
	tr := make([]float64, len(closes))

	for i := range closes {
		if i == 0 {
			tr[i] = highs[i] - lows[i]
			result[i] = math.NaN()
			continue
		}

		h_l := highs[i] - lows[i]
		h_pc := math.Abs(highs[i] - closes[i-1])
		l_pc := math.Abs(lows[i] - closes[i-1])

		tr[i] = math.Max(h_l, math.Max(h_pc, l_pc))

		if i < period {
			result[i] = math.NaN()
			continue
		}

		if i == period {
			sum := 0.0
			for j := 1; j <= period; j++ {
				sum += tr[j]
			}
			result[i] = sum / float64(period)
		} else {
			result[i] = (result[i-1]*float64(period-1) + tr[i]) / float64(period)
		}
	}

	return result
}

// calculateADX calculates the Average Directional Index
// calculateADX 计算平均趋势指数（趋势强度）
// ADX < 20: 无趋势，观望 / No trend, wait
// ADX 20-25: 弱趋势 / Weak trend
// ADX > 25: 强趋势，可交易 / Strong trend, tradable
// ADX > 50: 极强趋势，最佳机会 / Very strong trend, best opportunity
func calculateADX(highs, lows, closes []float64, period int) (adx, diPlus, diMinus []float64) {
	n := len(closes)
	adx = make([]float64, n)
	diPlus = make([]float64, n)
	diMinus = make([]float64, n)

	// Calculate True Range and Directional Movement
	// 计算真实波动幅度和趋向变动
	tr := make([]float64, n)
	plusDM := make([]float64, n)
	minusDM := make([]float64, n)

	for i := range closes {
		if i == 0 {
			tr[i] = highs[i] - lows[i]
			plusDM[i] = 0
			minusDM[i] = 0
			adx[i] = math.NaN()
			diPlus[i] = math.NaN()
			diMinus[i] = math.NaN()
			continue
		}

		// True Range
		h_l := highs[i] - lows[i]
		h_pc := math.Abs(highs[i] - closes[i-1])
		l_pc := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(h_l, math.Max(h_pc, l_pc))

		// Directional Movement
		upMove := highs[i] - highs[i-1]
		downMove := lows[i-1] - lows[i]

		if upMove > downMove && upMove > 0 {
			plusDM[i] = upMove
		} else {
			plusDM[i] = 0
		}

		if downMove > upMove && downMove > 0 {
			minusDM[i] = downMove
		} else {
			minusDM[i] = 0
		}

		if i < period {
			adx[i] = math.NaN()
			diPlus[i] = math.NaN()
			diMinus[i] = math.NaN()
		}
	}

	// Smooth True Range and Directional Movements
	// 平滑真实波动幅度和趋向变动
	smoothedTR := make([]float64, n)
	smoothedPlusDM := make([]float64, n)
	smoothedMinusDM := make([]float64, n)

	// Initial smoothing - sum of first period values
	// 初始平滑 - 第一个周期的总和
	for i := 1; i <= period && i < n; i++ {
		smoothedTR[period] += tr[i]
		smoothedPlusDM[period] += plusDM[i]
		smoothedMinusDM[period] += minusDM[i]
	}

	// Subsequent values use exponential smoothing
	// 后续值使用指数平滑
	for i := period + 1; i < n; i++ {
		smoothedTR[i] = smoothedTR[i-1] - (smoothedTR[i-1] / float64(period)) + tr[i]
		smoothedPlusDM[i] = smoothedPlusDM[i-1] - (smoothedPlusDM[i-1] / float64(period)) + plusDM[i]
		smoothedMinusDM[i] = smoothedMinusDM[i-1] - (smoothedMinusDM[i-1] / float64(period)) + minusDM[i]
	}

	// Calculate +DI and -DI
	// 计算 +DI 和 -DI
	dx := make([]float64, n)
	for i := period; i < n; i++ {
		if smoothedTR[i] != 0 {
			diPlus[i] = 100 * smoothedPlusDM[i] / smoothedTR[i]
			diMinus[i] = 100 * smoothedMinusDM[i] / smoothedTR[i]

			// Calculate DX
			diSum := diPlus[i] + diMinus[i]
			if diSum != 0 {
				dx[i] = 100 * math.Abs(diPlus[i]-diMinus[i]) / diSum
			} else {
				dx[i] = 0
			}
		} else {
			diPlus[i] = 0
			diMinus[i] = 0
			dx[i] = 0
		}
	}

	// Calculate ADX (smoothed DX)
	// 计算 ADX（平滑的 DX）
	adxPeriod := period // Use same period as DI (Wilder's standard method)
	for i := period + adxPeriod - 1; i < n; i++ {
		if i == period+adxPeriod-1 {
			// Initial ADX is average of first period DX values
			sum := 0.0
			for j := period; j < period+adxPeriod; j++ {
				sum += dx[j]
			}
			adx[i] = sum / float64(adxPeriod)
		} else {
			// Smooth ADX
			adx[i] = (adx[i-1]*float64(adxPeriod-1) + dx[i]) / float64(adxPeriod)
		}
	}

	return adx, diPlus, diMinus
}

// calculateVWAP calculates a rolling Volume Weighted Average Price over a fixed window
// windowBars is typically the number of candles in 24h for the current timeframe (e.g. 96 for 15m)
// calculateVWAP 计算固定窗口的滚动成交量加权平均价
// windowBars 通常是当前时间周期下 24 小时包含的 K 线数量（例如 15m = 96 根）
func calculateVWAP(highs, lows, closes, volumes []float64, windowBars int) []float64 {
	n := len(closes)
	result := make([]float64, n)

	// 基本安全检查：长度必须一致且非空
	if n == 0 || len(highs) != n || len(lows) != n || len(volumes) != n {
		for i := range result {
			result[i] = math.NaN()
		}
		return result
	}

	if windowBars <= 0 {
		windowBars = n
	}

	// Use prefix sums for O(1) window aggregation
	// 使用前缀和实现 O(1) 的窗口聚合
	cumPV := make([]float64, n+1)
	cumVol := make([]float64, n+1)

	for i := 0; i < n; i++ {
		// 使用典型价 (High+Low+Close)/3 作为价格代表
		typicalPrice := (highs[i] + lows[i] + closes[i]) / 3.0
		vol := volumes[i]
		pv := typicalPrice * vol

		cumPV[i+1] = cumPV[i] + pv
		cumVol[i+1] = cumVol[i] + vol

		// Sliding window [start, i]
		// 滚动窗口 [start, i]
		start := 0
		if i+1 > windowBars {
			start = i + 1 - windowBars
		}

		windowPV := cumPV[i+1] - cumPV[start]
		windowVol := cumVol[i+1] - cumVol[start]
		if windowVol == 0 {
			result[i] = math.NaN()
		} else {
			result[i] = windowPV / windowVol
		}
	}

	return result
}

// calculateVolumeRatio calculates volume ratio compared to average
// calculateVolumeRatio 计算成交量比率（相对于平均值）
// Ratio > 1.5: 放量 / High volume
// Ratio > 2.0: 异常放量 / Exceptionally high volume
func calculateVolumeRatio(volumes []float64, period int) []float64 {
	result := make([]float64, len(volumes))

	for i := range volumes {
		if i < period-1 {
			result[i] = math.NaN()
			continue
		}

		// Calculate average volume for the period
		// 计算周期内的平均成交量
		sum := 0.0
		for j := 0; j < period; j++ {
			sum += volumes[i-j]
		}
		avgVolume := sum / float64(period)

		// Calculate ratio
		// 计算比率
		if avgVolume > 0 {
			result[i] = volumes[i] / avgVolume
		} else {
			result[i] = 1.0
		}
	}

	return result
}

// FormatOHLCVReport generates a formatted report of OHLCV data
func FormatOHLCVReport(symbol string, timeframe string, ohlcvData []OHLCV) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Crypto data for %s\n", symbol))
	sb.WriteString(fmt.Sprintf("# Timeframe: %s\n", timeframe))
	sb.WriteString(fmt.Sprintf("# Total records: %d\n", len(ohlcvData)))
	sb.WriteString(fmt.Sprintf("# Data retrieved on: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	if len(ohlcvData) > 0 {
		sb.WriteString(fmt.Sprintf("# Latest data: %s\n",
			ohlcvData[len(ohlcvData)-1].Timestamp.Format("2006-01-02 15:04:05")))
	}
	sb.WriteString("\n")

	// Add CSV header
	sb.WriteString("timestamp,open,high,low,close,volume\n")

	// Add data - limit to last 100 candles to avoid context overflow
	startIdx := 0
	if len(ohlcvData) > 100 {
		startIdx = len(ohlcvData) - 100
	}

	for i := startIdx; i < len(ohlcvData); i++ {
		candle := ohlcvData[i]
		sb.WriteString(fmt.Sprintf("%s,%.2f,%.2f,%.2f,%.2f,%.2f\n",
			candle.Timestamp.Format("2006-01-02 15:04:05"),
			candle.Open,
			candle.High,
			candle.Low,
			candle.Close,
			candle.Volume,
		))
	}

	return sb.String()
}

// FormatIndicatorReport generates a formatted report of technical indicators
// 生成技术指标的格式化报告（日内数据）
func FormatIndicatorReport(symbol string, timeframe string, ohlcvData []OHLCV, indicators *TechnicalIndicators) string {
	var sb strings.Builder

	if len(ohlcvData) == 0 {
		sb.WriteString("无数据可用 (No data available)\n")
		return sb.String()
	}

	lastIdx := len(ohlcvData) - 1
	latestClosePrice := ohlcvData[lastIdx].Close

	// === 标题 ===
	// === Header ===
	sb.WriteString(fmt.Sprintf("=== %s Market Report ===\n\n", symbol))

	// === 当前值摘要 ===
	// === Current Values Summary ===
	currentEMA12 := 0.0
	if len(indicators.EMA_12) > lastIdx && !math.IsNaN(indicators.EMA_12[lastIdx]) {
		currentEMA12 = indicators.EMA_12[lastIdx]
	}

	currentEMA26 := 0.0
	if len(indicators.EMA_26) > lastIdx && !math.IsNaN(indicators.EMA_26[lastIdx]) {
		currentEMA26 = indicators.EMA_26[lastIdx]
	}

	currentMACD := 0.0
	if len(indicators.MACD) > lastIdx && !math.IsNaN(indicators.MACD[lastIdx]) {
		currentMACD = indicators.MACD[lastIdx]
	}

	// 当前 24 小时 VWAP（基于滚动窗口）
	// Current 24h VWAP based on rolling window
	currentVWAP := math.NaN()
	if len(indicators.VWAP) > lastIdx && !math.IsNaN(indicators.VWAP[lastIdx]) {
		currentVWAP = indicators.VWAP[lastIdx]
	}

	//currentMACDSignal := 0.0
	//if len(indicators.Signal) > lastIdx && !math.IsNaN(indicators.Signal[lastIdx]) {
	//	currentMACDSignal = indicators.Signal[lastIdx]
	//}

	currentRSI7 := 0.0
	if len(indicators.RSI_7) > lastIdx && !math.IsNaN(indicators.RSI_7[lastIdx]) {
		currentRSI7 = indicators.RSI_7[lastIdx]
	}

	currentRSI14 := 0.0
	if len(indicators.RSI) > lastIdx && !math.IsNaN(indicators.RSI[lastIdx]) {
		currentRSI14 = indicators.RSI[lastIdx]
	}

	currentADX := 0.0
	if len(indicators.ADX) > lastIdx && !math.IsNaN(indicators.ADX[lastIdx]) {
		currentADX = indicators.ADX[lastIdx]
	}

	// 构造 VWAP(24h滚动) 文本，仅在可用时追加
	// Build VWAP(24h rolling) text, append only when available
	vwapText := ""
	if !math.IsNaN(currentVWAP) && currentVWAP > 0 {
		diffPct := (latestClosePrice - currentVWAP) / currentVWAP * 100
		vwapText = fmt.Sprintf(", VWAP(24h滚动) = %.1f (价格较VWAP(24h滚动) %+0.2f%%)", currentVWAP, diffPct)
	}

	sb.WriteString(fmt.Sprintf("当前价格 = %.1f, EMA(12) = %.1f, EMA(26) = %.1f%s\n", latestClosePrice, currentEMA12, currentEMA26, vwapText))
	sb.WriteString(fmt.Sprintf("MACD = %.1f,  RSI(7) = %.1f, RSI(14) = %.1f, ADX = %.1f\n\n", currentMACD, currentRSI7, currentRSI14, currentADX))
	// 说明 VWAP(24h滚动) 的定义，避免被误解为「开盘以来 VWAP」
	// Explain VWAP(24h rolling) definition to avoid confusion with intraday VWAP
	sb.WriteString("说明：本报告中的 VWAP(24h滚动)，是过去连续24小时的成交量加权平均价（跨日滚动计算），不是“今天开盘到现在”的日内 VWAP。\n\n")
	sb.WriteString("下述所有价格或信号数据均按时间从旧到新排列。\n\n")

	// === 日内数据（最近10期）===
	// === Intraday Data (Last 10 periods) ===
	sb.WriteString(fmt.Sprintf("日内数据(序列为%s间隔):\n\n", timeframe))

	// Determine series length (up to 10 data points)
	// 确定序列长度（最多10个数据点）
	seriesLength := 10
	startIdx := lastIdx - seriesLength + 1
	if startIdx < 0 {
		startIdx = 0
	}

	// Helper function to format float array (last N values)
	// 辅助函数：格式化浮点数数组（最近 N 个值）
	formatSeries := func(data []float64, startIdx, endIdx int, decimals int) string {
		var values []string
		for i := startIdx; i <= endIdx; i++ {
			if i >= 0 && i < len(data) && !math.IsNaN(data[i]) {
				values = append(values, fmt.Sprintf("%.*f", decimals, data[i]))
			}
		}
		return "[" + strings.Join(values, ", ") + "]"
	}

	// 1. 中间价及相对 VWAP(24h滚动) 偏离% 序列
	// 1. Mid prices and deviations from VWAP(24h rolling) in percentage
	var midPrices []float64
	for i := startIdx; i <= lastIdx; i++ {
		midPrice := (ohlcvData[i].High + ohlcvData[i].Low) / 2
		midPrices = append(midPrices, midPrice)
	}

	if !math.IsNaN(currentVWAP) && currentVWAP > 0 {
		// 同时打印中间价与相对 VWAP(24h滚动) 的偏离百分比，格式为 price:dev%%
		// Print mid price and deviation from VWAP(24h rolling) together as price:dev%%
		var pairs []string
		for _, price := range midPrices {
			dev := (price - currentVWAP) / currentVWAP * 100
			pairs = append(pairs, fmt.Sprintf("%.1f:%.2f", price, dev))
		}
		// 注意：这里使用 "%%" 在 fmt.Sprintf 中输出一个真实的百分号 "%"
		// Note: use "%%" in fmt.Sprintf format string to output a literal "%" character
		sb.WriteString(fmt.Sprintf("中间价及相对 VWAP(24h滚动) 偏离%%(%s间隔): [%s]\n\n", timeframe, strings.Join(pairs, ", ")))
	} else {
		// 若当前 VWAP 不可用，仅输出中间价序列
		// If current VWAP is not available, only output mid price series
		sb.WriteString(fmt.Sprintf("中间价(%s间隔): %s\n\n", timeframe, formatSeries(midPrices, 0, len(midPrices)-1, 1)))
	}

	// 2. EMA(12) + EMA(26) 快慢EMA系统（MACD基础）
	// EMA(12) + EMA(26) Fast/Slow EMA System (MACD basis: MACD = EMA12 - EMA26)
	if len(indicators.EMA_12) > lastIdx {
		sb.WriteString(fmt.Sprintf("EMA(12): %s\n\n", formatSeries(indicators.EMA_12, startIdx, lastIdx, 1)))
	}
	if len(indicators.EMA_26) > lastIdx {
		sb.WriteString(fmt.Sprintf("EMA(26): %s\n\n", formatSeries(indicators.EMA_26, startIdx, lastIdx, 1)))
	}

	// 3. MACD + MACD_Signal 趋势动能 + 交叉信号
	// MACD + MACD_Signal: Trend Momentum + Crossover Signal
	// 金叉(Golden Cross): MACD上穿MACD_Signal → 买入信号
	// 死叉(Death Cross): MACD下穿MACD_Signal → 卖出信号
	if len(indicators.MACD) > lastIdx {
		sb.WriteString(fmt.Sprintf("MACD: %s\n\n", formatSeries(indicators.MACD, startIdx, lastIdx, 1)))
	}
	//if len(indicators.Signal) > lastIdx {
	//	sb.WriteString(fmt.Sprintf("MACD-DEA: %s\n\n", formatSeries(indicators.Signal, startIdx, lastIdx, 1)))
	//}

	// 4. BB_Upper + BB_Lower 波动率通道
	// BB_Upper + BB_Lower Volatility Bands
	if len(indicators.BB_Upper) > lastIdx {
		sb.WriteString(fmt.Sprintf("BB_Upper: %s\n\n", formatSeries(indicators.BB_Upper, startIdx, lastIdx, 1)))
	}
	if len(indicators.BB_Lower) > lastIdx {
		sb.WriteString(fmt.Sprintf("BB_Lower: %s\n\n", formatSeries(indicators.BB_Lower, startIdx, lastIdx, 1)))
	}

	// 5. RSI(7) + RSI(14) 短期+标准超买超卖
	// RSI(7) + RSI(14) Short-term + Standard Overbought/Oversold
	if len(indicators.RSI_7) > lastIdx {
		sb.WriteString(fmt.Sprintf("RSI(7): %s\n\n", formatSeries(indicators.RSI_7, startIdx, lastIdx, 1)))
	}
	if len(indicators.RSI) > lastIdx {
		sb.WriteString(fmt.Sprintf("RSI(14): %s\n\n", formatSeries(indicators.RSI, startIdx, lastIdx, 1)))
	}

	// 6. ADX 趋势强度过滤器
	// ADX Trend Strength Filter
	if len(indicators.ADX) > lastIdx {
		sb.WriteString(fmt.Sprintf("ADX: %s\n\n", formatSeries(indicators.ADX, startIdx, lastIdx, 1)))
	}

	return sb.String()
}

// GetFundingRate fetches the current funding rate
func (m *MarketData) GetFundingRate(ctx context.Context, symbol string) (float64, error) {
	rates, err := m.client.NewFundingRateService().
		Symbol(symbol).
		Limit(1).
		Do(ctx)

	if err != nil {
		return 0, fmt.Errorf("failed to fetch funding rate: %w", err)
	}

	if len(rates) == 0 {
		return 0, fmt.Errorf("no funding rate data available")
	}

	fundingRate, _ := strconv.ParseFloat(rates[0].FundingRate, 64)
	return fundingRate, nil
}

// GetOrderBook fetches the order book depth
func (m *MarketData) GetOrderBook(ctx context.Context, symbol string, limit int) (map[string]interface{}, error) {
	depth, err := m.client.NewDepthService().
		Symbol(symbol).
		Limit(limit).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch order book: %w", err)
	}

	// Calculate bid/ask strength
	var bidVolume, askVolume float64
	for _, bid := range depth.Bids {
		qty, _ := strconv.ParseFloat(bid.Quantity, 64)
		bidVolume += qty
	}
	for _, ask := range depth.Asks {
		qty, _ := strconv.ParseFloat(ask.Quantity, 64)
		askVolume += qty
	}

	result := map[string]interface{}{
		"bids":          depth.Bids,
		"asks":          depth.Asks,
		"bid_volume":    bidVolume,
		"ask_volume":    askVolume,
		"bid_ask_ratio": bidVolume / (askVolume + 0.0001),
	}

	return result, nil
}

// Get24HrStats fetches 24-hour statistics
func (m *MarketData) Get24HrStats(ctx context.Context, symbol string) (map[string]string, error) {
	stats, err := m.client.NewListPriceChangeStatsService().
		Symbol(symbol).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch 24hr stats: %w", err)
	}

	if len(stats) == 0 {
		return nil, fmt.Errorf("no stats data available")
	}

	result := map[string]string{
		"price_change":         stats[0].PriceChange,
		"price_change_percent": stats[0].PriceChangePercent,
		"high_price":           stats[0].HighPrice,
		"low_price":            stats[0].LowPrice,
		"volume":               stats[0].Volume,
		"quote_volume":         stats[0].QuoteVolume,
	}

	return result, nil
}

// GetOpenInterest fetches the current open interest data
// GetOpenInterest 获取当前未平仓合约数据
func (m *MarketData) GetOpenInterest(ctx context.Context, symbol string) (map[string]float64, error) {
	// Get current open interest
	openInterest, err := m.client.NewGetOpenInterestService().
		Symbol(symbol).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch open interest: %w", err)
	}

	currentOI, _ := strconv.ParseFloat(openInterest.OpenInterest, 64)

	// Get historical open interest statistics (for average calculation)
	// 获取历史未平仓数据统计（用于计算平均值）
	histStats, err := m.client.NewOpenInterestStatisticsService().
		Symbol(symbol).
		Period("5m").
		Limit(12). // Last 12 periods (1 hour if 5m intervals)
		Do(ctx)

	var avgOI float64
	if err == nil && len(histStats) > 0 {
		var sum float64
		for _, stat := range histStats {
			oi, _ := strconv.ParseFloat(stat.SumOpenInterest, 64)
			sum += oi
		}
		avgOI = sum / float64(len(histStats))
	} else {
		avgOI = currentOI // Fallback to current if historical data unavailable
	}

	result := map[string]float64{
		"latest":  currentOI,
		"average": avgOI,
	}

	return result, nil
}

// GetTopLongShortPositionRatio 获取大户持仓多空比（支持 1h 和 4h 周期）
// GetTopLongShortPositionRatio gets top trader long/short position ratio for specified period
func (m *MarketData) GetTopLongShortPositionRatio(ctx context.Context, symbol string, period string, limit int) (map[string]interface{}, error) {
	ratios, err := m.client.NewTopLongShortPositionRatioService().
		Symbol(symbol).
		Period(period).
		Limit(uint32(limit)).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch top long/short position ratio: %w", err)
	}

	if len(ratios) == 0 {
		return nil, fmt.Errorf("no data returned for top long/short position ratio")
	}

	// Binance API returns data in oldest-to-newest order (same as OpenInterestStatistics)
	// 币安 API 返回数据按从旧到新的顺序（与 OpenInterestStatistics 相同）
	// ratios[0] = oldest, ratios[len-1] = newest
	// ratios[0] = 最旧，ratios[len-1] = 最新

	// Prepare series data (API already returns oldest to newest, no need to reverse)
	// 准备序列数据（API 已经按从旧到新返回，无需反转）
	seriesRatios := make([]float64, 0, len(ratios))
	for i := 0; i < len(ratios); i++ {
		value, err := strconv.ParseFloat(ratios[i].LongShortRatio, 64)
		if err != nil {
			continue
		}
		seriesRatios = append(seriesRatios, value)
	}

	// Get the latest data point (last element in array)
	// 获取最新数据点（数组最后一个元素）
	latest := ratios[len(ratios)-1]
	longShortRatio, _ := strconv.ParseFloat(latest.LongShortRatio, 64)
	longAccount, _ := strconv.ParseFloat(latest.LongAccount, 64)
	shortAccount, _ := strconv.ParseFloat(latest.ShortAccount, 64)

	result := map[string]interface{}{
		"period":           period,
		"long_short_ratio": longShortRatio,
		"long_account":     longAccount * 100,  // Convert to percentage
		"short_account":    shortAccount * 100, // Convert to percentage
		"timestamp":        latest.Timestamp,
		"series_ratios":    seriesRatios,
	}

	return result, nil
}

// GetVWAPDeviationHistory 获取 VWAP 历史偏离数据（价格序列和偏离百分比）
// GetVWAPDeviationHistory gets VWAP deviation history (price series and deviation percentages)
func (m *MarketData) GetVWAPDeviationHistory(ctx context.Context, symbol string, timeframe string, lookbackPeriods int) (map[string]interface{}, error) {
	// Get OHLCV data for the specified timeframe
	// 获取指定时间框架的 OHLCV 数据
	lookbackDays := 2 // 2 days should be enough for most timeframes
	ohlcvData, err := m.GetOHLCV(ctx, symbol, timeframe, lookbackDays)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OHLCV data: %w", err)
	}

	if len(ohlcvData) < lookbackPeriods {
		return nil, fmt.Errorf("insufficient data: got %d periods, need %d", len(ohlcvData), lookbackPeriods)
	}

	// Calculate VWAP for 24-hour rolling window
	// 计算 24 小时滚动窗口的 VWAP
	indicators := CalculateIndicators(ohlcvData)
	if len(indicators.VWAP) == 0 {
		return nil, fmt.Errorf("failed to calculate VWAP")
	}

	// Get the last N periods
	// 获取最近 N 个周期
	startIdx := len(ohlcvData) - lookbackPeriods
	if startIdx < 0 {
		startIdx = 0
	}

	priceHistory := make([]float64, 0, lookbackPeriods)
	deviationHistory := make([]float64, 0, lookbackPeriods)

	// Extract price and deviation data
	// 提取价格和偏离数据
	for i := startIdx; i < len(ohlcvData); i++ {
		// Use mid price (average of high and low)
		// 使用中间价（最高价和最低价的平均值）
		midPrice := (ohlcvData[i].High + ohlcvData[i].Low) / 2
		priceHistory = append(priceHistory, midPrice)

		// Calculate deviation from VWAP
		// 计算相对于 VWAP 的偏离
		if i < len(indicators.VWAP) && !math.IsNaN(indicators.VWAP[i]) && indicators.VWAP[i] > 0 {
			deviation := ((midPrice - indicators.VWAP[i]) / indicators.VWAP[i]) * 100
			deviationHistory = append(deviationHistory, deviation)
		} else {
			deviationHistory = append(deviationHistory, 0.0)
		}
	}

	// Get current VWAP value
	// 获取当前 VWAP 值
	currentVWAP := 0.0
	if len(indicators.VWAP) > 0 && !math.IsNaN(indicators.VWAP[len(indicators.VWAP)-1]) {
		currentVWAP = indicators.VWAP[len(indicators.VWAP)-1]
	}

	// Calculate current deviation
	// 计算当前偏离
	currentDeviation := 0.0
	if len(ohlcvData) > 0 && currentVWAP > 0 {
		currentPrice := (ohlcvData[len(ohlcvData)-1].High + ohlcvData[len(ohlcvData)-1].Low) / 2
		currentDeviation = ((currentPrice - currentVWAP) / currentVWAP) * 100
	}

	result := map[string]interface{}{
		"vwap_24h":          currentVWAP,
		"current_deviation": currentDeviation,
		"price_history":     priceHistory,
		"deviation_history": deviationHistory,
	}

	return result, nil
}

// GetOpenInterestChange 获取持仓量变化统计（对比当前和历史数据）
// GetOpenInterestChange gets open interest change by comparing current and historical data
func (m *MarketData) GetOpenInterestChange(ctx context.Context, symbol string, period string, limit int) (map[string]interface{}, error) {
	stats, err := m.client.NewOpenInterestStatisticsService().
		Symbol(symbol).
		Period(period).
		Limit(limit).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch open interest statistics: %w", err)
	}

	if len(stats) == 0 {
		return nil, fmt.Errorf("no data returned for open interest statistics")
	}

	// Binance API returns data in oldest-to-newest order (confirmed by timestamp comparison)
	// 币安 API 返回数据按从旧到新的顺序（通过时间戳对比确认）
	// stats[0] = oldest, stats[len-1] = newest
	// stats[0] = 最旧，stats[len-1] = 最新

	// Calculate change if we have at least 2 data points
	// 如果有至少 2 个数据点，计算变化率
	// Use the newest data point as "current"
	// 使用最新的数据点作为"当前值"
	lastIdx := len(stats) - 1
	current, _ := strconv.ParseFloat(stats[lastIdx].SumOpenInterestValue, 64)
	currentOI, _ := strconv.ParseFloat(stats[lastIdx].SumOpenInterest, 64)

	var changePercent float64
	var previous float64

	if len(stats) >= 2 {
		previous, _ = strconv.ParseFloat(stats[lastIdx-1].SumOpenInterestValue, 64)
		if previous > 0 {
			changePercent = ((current - previous) / previous) * 100
		}
	}

	// Build chronological series data (oldest to newest)
	// 构建时间顺序的序列数据（从旧到新）
	// API already returns data in chronological order, so no need to reverse
	// API 已经按时间顺序返回数据，无需反转
	seriesValues := make([]float64, 0, len(stats))
	seriesVolumes := make([]float64, 0, len(stats))
	changeRates := make([]float64, 0, len(stats))

	for i := 0; i < len(stats); i++ {
		value, err := strconv.ParseFloat(stats[i].SumOpenInterestValue, 64)
		if err != nil {
			continue
		}
		volume, err := strconv.ParseFloat(stats[i].SumOpenInterest, 64)
		if err != nil {
			continue
		}

		seriesValues = append(seriesValues, value)
		seriesVolumes = append(seriesVolumes, volume)

		// Calculate change rate relative to previous point
		// 计算相对于上一个点的变化率
		if i == 0 {
			changeRates = append(changeRates, 0.0) // First point has no previous, set to 0
		} else {
			prevValue, _ := strconv.ParseFloat(stats[i-1].SumOpenInterest, 64)
			if prevValue > 0 {
				changeRate := ((volume - prevValue) / prevValue) * 100
				changeRates = append(changeRates, changeRate)
			} else {
				changeRates = append(changeRates, 0.0)
			}
		}
	}

	result := map[string]interface{}{
		"period":            period,
		"current_oi_value":  current,
		"current_oi":        currentOI,
		"previous_oi_value": previous,
		"change_percent":    changePercent,
		"timestamp":         stats[lastIdx].Timestamp, // Use newest timestamp / 使用最新时间戳
		"series_values":     seriesValues,
		"series_volumes":    seriesVolumes, // 持仓量序列 / Open interest volume series
		"change_rates":      changeRates,   // 变化率序列 / Change rate series
	}

	return result, nil
}

// FormatOrderBookReport formats order book data into a detailed report for LLM
// FormatOrderBookReport 将订单簿数据格式化为 LLM 易读的详细报告
func FormatOrderBookReport(orderBook map[string]interface{}, topN int) string {
	var report strings.Builder

	bidVolume := orderBook["bid_volume"].(float64)
	askVolume := orderBook["ask_volume"].(float64)
	bidAskRatio := orderBook["bid_ask_ratio"].(float64)

	report.WriteString(fmt.Sprintf("📊 当前订单簿深度分析（前 %d 档）:\n", topN))
	report.WriteString(fmt.Sprintf("  买卖盘总量: 买 %.2f vs 卖 %.2f\n", bidVolume, askVolume))
	report.WriteString(fmt.Sprintf("  买卖比: %.2f\n", bidAskRatio))

	return report.String()
}

// Helper functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func formatPrice(priceStr string) string {
	price, err := strconv.ParseFloat(priceStr, 64)
	if err != nil {
		return priceStr
	}

	// Format with appropriate decimals based on price magnitude
	if price >= 1000 {
		return fmt.Sprintf("%.2f", price)
	} else if price >= 1 {
		return fmt.Sprintf("%.4f", price)
	} else {
		return fmt.Sprintf("%.6f", price)
	}
}

func convertTimeframe(tf string) string {
	// Convert from format like "1h", "15m", "1d" to Binance interval format
	// Binance supports: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w, 1M
	// 币安支持的时间周期：1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w, 1M
	switch tf {
	case "1m", "3m", "5m", "15m", "30m": // 分钟级
		return tf
	case "1h", "2h", "4h", "6h", "8h", "12h": // 小时级
		return tf
	case "1d", "3d": // 天级
		return tf
	case "1w": // 周级
		return tf
	case "1M": // 月级
		return tf
	default:
		// 不支持的时间周期，返回默认值 1h
		// Unsupported timeframe, return default 1h
		return "1h"
	}
}

// FormatLongerTimeframeReport generates a formatted report for longer timeframe analysis
// FormatLongerTimeframeReport 生成更长期时间周期分析的格式化报告
func FormatLongerTimeframeReport(symbol string, timeframe string, ohlcvData []OHLCV, indicators *TechnicalIndicators) string {
	var sb strings.Builder

	if len(ohlcvData) == 0 {
		sb.WriteString("无数据可用 (No data available)\n")
		return sb.String()
	}

	lastIdx := len(ohlcvData) - 1

	// === 长期数据标题 ===
	// === Long-term Data Header ===
	sb.WriteString(fmt.Sprintf("长期数据 (序列为%s间隔):\n", timeframe))

	// === 序列数据配置 ===
	// === Series Data Configuration ===
	seriesLength := 10
	startIdx := lastIdx - seriesLength + 1
	if startIdx < 0 {
		startIdx = 0
	}

	// Helper function to format float array (last N values)
	// 辅助函数：格式化浮点数数组（最近 N 个值）
	formatSeries := func(data []float64, startIdx, endIdx int, decimals int) string {
		var values []string
		for i := startIdx; i <= endIdx; i++ {
			if i >= 0 && i < len(data) && !math.IsNaN(data[i]) {
				values = append(values, fmt.Sprintf("%.*f", decimals, data[i]))
			}
		}
		return "[" + strings.Join(values, ", ") + "]"
	}

	// === 中间价序列（最近10期）===
	// === Middle Price Series (Last 10 periods) ===
	var middlePrices []string
	for i := startIdx; i <= lastIdx; i++ {
		if i >= 0 && i < len(ohlcvData) {
			middlePrice := (ohlcvData[i].High + ohlcvData[i].Low) / 2
			middlePrices = append(middlePrices, fmt.Sprintf("%.1f", middlePrice))
		}
	}
	sb.WriteString(fmt.Sprintf("中间价: [%s]\n", strings.Join(middlePrices, ", ")))

	// === EMA(20) vs 50-Period EMA ===
	ema20Val := 0.0
	ema50Val := 0.0
	if len(indicators.EMA_20) > lastIdx && !math.IsNaN(indicators.EMA_20[lastIdx]) {
		ema20Val = indicators.EMA_20[lastIdx]
	}
	if len(indicators.EMA_50) > lastIdx && !math.IsNaN(indicators.EMA_50[lastIdx]) {
		ema50Val = indicators.EMA_50[lastIdx]
	}
	sb.WriteString(fmt.Sprintf("EMA(20): %.1f vs. EMA(50): %.1f\n\n", ema20Val, ema50Val))

	// === ATR(3) vs ATR(7) vs ATR(14) ===
	atr3Val := 0.0
	atr7Val := 0.0
	atr14Val := 0.0

	if len(indicators.ATR_3) > lastIdx && !math.IsNaN(indicators.ATR_3[lastIdx]) {
		atr3Val = indicators.ATR_3[lastIdx]
	}
	if len(indicators.ATR_7) > lastIdx && !math.IsNaN(indicators.ATR_7[lastIdx]) {
		atr7Val = indicators.ATR_7[lastIdx]
	}
	if len(indicators.ATR_14) > lastIdx && !math.IsNaN(indicators.ATR_14[lastIdx]) {
		atr14Val = indicators.ATR_14[lastIdx]
	}
	sb.WriteString(fmt.Sprintf("ATR(3): %.1f vs. ATR(7): %.1f vs. ATR(14): %.1f\n\n", atr3Val, atr7Val, atr14Val))

	// === 当前成交量 vs 平均成交量 ===
	// === Current Volume vs Average Volume ===
	currentVolume := 0.0
	avgVolume := 0.0
	if len(ohlcvData) >= 20 {
		currentVolume = ohlcvData[lastIdx].Volume
		for i := lastIdx - 19; i <= lastIdx; i++ {
			avgVolume += ohlcvData[i].Volume
		}
		avgVolume /= 20
	}
	sb.WriteString(fmt.Sprintf("当前成交量: %.1f vs. 平均成交量: %.1f\n\n", currentVolume, avgVolume))

	// === MACD 序列（最近10期）===
	// === MACD Series (Last 10 periods) ===
	if len(indicators.MACD) > lastIdx {
		sb.WriteString(fmt.Sprintf("MACD: %s\n\n", formatSeries(indicators.MACD, startIdx, lastIdx, 1)))
	}

	// === RSI(14) 序列（最近10期）===
	// === RSI(14) Series (Last 10 periods) ===
	if len(indicators.RSI) > lastIdx {
		sb.WriteString(fmt.Sprintf("RSI(14): %s\n\n", formatSeries(indicators.RSI, startIdx, lastIdx, 1)))
	}

	return sb.String()
}

// GetMultiTimeframeIndicators fetches and calculates indicators for multiple timeframes in parallel
// GetMultiTimeframeIndicators 并行获取多个时间框架的数据并计算指标
func (m *MarketData) GetMultiTimeframeIndicators(ctx context.Context, symbol string) []MultiTimeframeIndicator {
	// Define fixed timeframes for multi-timeframe analysis
	// 定义固定的多时间框架列表（经典的多周期分析组合）
	timeframes := []string{"5m", "15m", "1h", "4h"}

	// Calculate lookback days for each timeframe
	// 为每个时间框架计算回看天数
	// 注意：币安API限制最多返回1000根K线
	lookbackDays := map[string]int{
		"5m":  3,  // ~864 candles (3天 × 24h × 60m / 5m = 864)
		"15m": 5,  // ~480 candles (5天 × 24h × 60m / 15m = 480)
		"1h":  10, // ~240 candles (10天 × 24h / 1h = 240)
		"4h":  15, // ~90 candles (15天 × 24h / 4h = 90)
	}

	// Use goroutines to fetch data in parallel
	// 使用 goroutine 并行获取数据
	var wg sync.WaitGroup
	results := make([]MultiTimeframeIndicator, len(timeframes))

	for i, tf := range timeframes {
		wg.Add(1)
		go func(index int, timeframe string) {
			defer wg.Done()

			// Get OHLCV data for this timeframe
			// 获取该时间框架的 OHLCV 数据
			lookback := lookbackDays[timeframe]
			ohlcvData, err := m.GetOHLCV(ctx, symbol, timeframe, lookback)
			if err != nil || len(ohlcvData) == 0 {
				// Return empty indicator on error
				// 出错时返回空指标
				results[index] = MultiTimeframeIndicator{
					Timeframe: timeframe,
					EMA20:     math.NaN(),
					EMA50:     math.NaN(),
					MACD:      math.NaN(),
					RSI7:      math.NaN(),
					RSI14:     math.NaN(),
				}
				return
			}

			// Calculate indicators
			// 计算技术指标
			indicators := CalculateIndicators(ohlcvData)

			// Extract the latest values
			// 提取最新值
			lastIdx := len(ohlcvData) - 1
			result := MultiTimeframeIndicator{
				Timeframe: timeframe,
				EMA20:     math.NaN(),
				EMA50:     math.NaN(),
				MACD:      math.NaN(),
				RSI7:      math.NaN(),
				RSI14:     math.NaN(),
			}

			if len(indicators.EMA_20) > lastIdx && !math.IsNaN(indicators.EMA_20[lastIdx]) {
				result.EMA20 = indicators.EMA_20[lastIdx]
			}
			if len(indicators.EMA_50) > lastIdx && !math.IsNaN(indicators.EMA_50[lastIdx]) {
				result.EMA50 = indicators.EMA_50[lastIdx]
			}
			if len(indicators.MACD) > lastIdx && !math.IsNaN(indicators.MACD[lastIdx]) {
				result.MACD = indicators.MACD[lastIdx]
			}
			if len(indicators.RSI_7) > lastIdx && !math.IsNaN(indicators.RSI_7[lastIdx]) {
				result.RSI7 = indicators.RSI_7[lastIdx]
			}
			if len(indicators.RSI) > lastIdx && !math.IsNaN(indicators.RSI[lastIdx]) {
				result.RSI14 = indicators.RSI[lastIdx]
			}

			results[index] = result
		}(i, tf)
	}

	wg.Wait()
	return results
}

// FormatMultiTimeframeReport generates a formatted report of multi-timeframe indicators
// FormatMultiTimeframeReport 生成多时间框架指标的格式化报告
func FormatMultiTimeframeReport(indicators []MultiTimeframeIndicator) string {
	var sb strings.Builder

	if len(indicators) == 0 {
		return ""
	}

	sb.WriteString("多时间框架指标：\n")

	// Define display names for timeframes (Chinese)
	// 定义时间框架的显示名称（中文）
	displayNames := map[string]string{
		"5m":  "5分钟",
		"15m": "15分钟",
		"1h":  "1小时",
		"4h":  "4小时",
	}

	for _, ind := range indicators {
		displayName := displayNames[ind.Timeframe]
		if displayName == "" {
			displayName = ind.Timeframe
		}

		// Format each indicator value (handle NaN cases)
		// 格式化每个指标值（处理 NaN 情况）
		ema20Str := "N/A"
		if !math.IsNaN(ind.EMA20) {
			ema20Str = fmt.Sprintf("%.3f", ind.EMA20)
		}

		ema50Str := "N/A"
		if !math.IsNaN(ind.EMA50) {
			ema50Str = fmt.Sprintf("%.3f", ind.EMA50)
		}

		macdStr := "N/A"
		if !math.IsNaN(ind.MACD) {
			macdStr = fmt.Sprintf("%.3f", ind.MACD)
		}

		rsi7Str := "N/A"
		if !math.IsNaN(ind.RSI7) {
			rsi7Str = fmt.Sprintf("%.2f", ind.RSI7)
		}

		rsi14Str := "N/A"
		if !math.IsNaN(ind.RSI14) {
			rsi14Str = fmt.Sprintf("%.2f", ind.RSI14)
		}

		// Format: "3分钟:  EMA20=86014.071, EMA50=86066.329, MACD=-28.439, RSI7=48.63, RSI14=51.64"
		sb.WriteString(fmt.Sprintf("%-6s  EMA20=%s, EMA50=%s, MACD=%s, RSI7=%s, RSI14=%s\n",
			displayName+":", ema20Str, ema50Str, macdStr, rsi7Str, rsi14Str))
	}

	return sb.String()
}

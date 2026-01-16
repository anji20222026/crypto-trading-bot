package dataflows

import (
	"math"
)

// SupportResistanceLevel represents a support or resistance level
// SupportResistanceLevel 表示支撑位或阻力位
type SupportResistanceLevel struct {
	Price      float64 // 价格 / Price
	Strength   int     // 强度（触及次数）/ Strength (number of touches)
	LastTouch  int     // 最后触及的索引 / Last touch index
	Type       string  // "support" or "resistance"
	IsRecent   bool    // 是否为最近验证过的 / Whether recently validated
	IsValid    bool    // 是否仍然有效 / Whether still valid
}

// CalculateSupportResistance calculates support and resistance levels from OHLCV data
// CalculateSupportResistance 从 OHLCV 数据计算支撑位和阻力位
//
// Parameters:
// 参数：
//   - highs: High prices / 最高价数组
//   - lows: Low prices / 最低价数组
//   - closes: Close prices / 收盘价数组
//   - lookback: Lookback period for swing detection / Swing 检测的回溯周期
//   - currentPrice: Current market price / 当前市场价格
//
// Returns:
// 返回：
//   - supportLevels: Array of support levels / 支撑位数组
//   - resistanceLevels: Array of resistance levels / 阻力位数组
func CalculateSupportResistance(
	highs []float64,
	lows []float64,
	closes []float64,
	lookback int,
	currentPrice float64,
) ([]float64, []float64) {
	if len(highs) < lookback*2+1 || len(lows) < lookback*2+1 {
		return []float64{}, []float64{}
	}

	// Find swing highs and lows
	// 查找摆动高点和低点
	swingHighs := findSwingHighs(highs, lookback)
	swingLows := findSwingLows(lows, lookback)

	// Cluster nearby levels
	// 聚类相近的价格水平
	supportClusters := clusterLevels(swingLows, currentPrice*0.005) // 0.5% tolerance
	resistanceClusters := clusterLevels(swingHighs, currentPrice*0.005)

	// Filter and sort levels
	// 过滤并排序价格水平
	supports := filterAndSortLevels(supportClusters, currentPrice, "support", 3)
	resistances := filterAndSortLevels(resistanceClusters, currentPrice, "resistance", 3)

	return supports, resistances
}

// findSwingHighs identifies swing high points
// findSwingHighs 识别摆动高点
//
// A swing high is a peak where the high is higher than N bars before and after
// 摆动高点是指高点高于前后 N 根 K 线的峰值
func findSwingHighs(highs []float64, lookback int) []SupportResistanceLevel {
	var swingHighs []SupportResistanceLevel

	for i := lookback; i < len(highs)-lookback; i++ {
		isSwingHigh := true
		currentHigh := highs[i]

		// Check if current high is higher than lookback bars before and after
		// 检查当前高点是否高于前后 lookback 根 K 线
		for j := 1; j <= lookback; j++ {
			if currentHigh <= highs[i-j] || currentHigh <= highs[i+j] {
				isSwingHigh = false
				break
			}
		}

		if isSwingHigh {
			swingHighs = append(swingHighs, SupportResistanceLevel{
				Price:     currentHigh,
				Strength:  1,
				LastTouch: i,
				Type:      "resistance",
				IsRecent:  i >= len(highs)-20, // Last 20 bars considered recent
				IsValid:   true,
			})
		}
	}

	return swingHighs
}

// findSwingLows identifies swing low points
// findSwingLows 识别摆动低点
//
// A swing low is a trough where the low is lower than N bars before and after
// 摆动低点是指低点低于前后 N 根 K 线的谷值
func findSwingLows(lows []float64, lookback int) []SupportResistanceLevel {
	var swingLows []SupportResistanceLevel

	for i := lookback; i < len(lows)-lookback; i++ {
		isSwingLow := true
		currentLow := lows[i]

		// Check if current low is lower than lookback bars before and after
		// 检查当前低点是否低于前后 lookback 根 K 线
		for j := 1; j <= lookback; j++ {
			if currentLow >= lows[i-j] || currentLow >= lows[i+j] {
				isSwingLow = false
				break
			}
		}

		if isSwingLow {
			swingLows = append(swingLows, SupportResistanceLevel{
				Price:     currentLow,
				Strength:  1,
				LastTouch: i,
				Type:      "support",
				IsRecent:  i >= len(lows)-20, // Last 20 bars considered recent
				IsValid:   true,
			})
		}
	}

	return swingLows
}

// clusterLevels groups nearby price levels together
// clusterLevels 将相近的价格水平聚类
func clusterLevels(levels []SupportResistanceLevel, tolerance float64) []SupportResistanceLevel {
	if len(levels) == 0 {
		return []SupportResistanceLevel{}
	}

	var clusters []SupportResistanceLevel

	for _, level := range levels {
		merged := false

		// Try to merge with existing cluster
		// 尝试与现有聚类合并
		for i := range clusters {
			if math.Abs(level.Price-clusters[i].Price) <= tolerance {
				// Merge: update price to weighted average, increase strength
				// 合并：更新价格为加权平均，增加强度
				totalStrength := clusters[i].Strength + level.Strength
				clusters[i].Price = (clusters[i].Price*float64(clusters[i].Strength) + level.Price*float64(level.Strength)) / float64(totalStrength)
				clusters[i].Strength = totalStrength
				if level.LastTouch > clusters[i].LastTouch {
					clusters[i].LastTouch = level.LastTouch
					clusters[i].IsRecent = level.IsRecent
				}
				merged = true
				break
			}
		}

		// If not merged, create new cluster
		// 如果未合并，创建新聚类
		if !merged {
			clusters = append(clusters, level)
		}
	}

	return clusters
}

// filterAndSortLevels filters levels by position relative to current price and sorts by proximity
// filterAndSortLevels 根据相对于当前价格的位置过滤价格水平，并按接近度排序
func filterAndSortLevels(
	levels []SupportResistanceLevel,
	currentPrice float64,
	levelType string,
	maxLevels int,
) []float64 {
	var filtered []SupportResistanceLevel

	// Filter by type and position
	// 按类型和位置过滤
	for _, level := range levels {
		if levelType == "support" && level.Price < currentPrice {
			// Support must be below current price
			// 支撑位必须在当前价格下方
			filtered = append(filtered, level)
		} else if levelType == "resistance" && level.Price > currentPrice {
			// Resistance must be above current price
			// 阻力位必须在当前价格上方
			filtered = append(filtered, level)
		}
	}

	// Sort by proximity to current price (closest first)
	// 按与当前价格的接近度排序（最近的优先）
	for i := 0; i < len(filtered); i++ {
		for j := i + 1; j < len(filtered); j++ {
			distI := math.Abs(filtered[i].Price - currentPrice)
			distJ := math.Abs(filtered[j].Price - currentPrice)
			if distJ < distI {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	// Limit to maxLevels
	// 限制到最大数量
	if len(filtered) > maxLevels {
		filtered = filtered[:maxLevels]
	}

	// Extract prices
	// 提取价格
	result := make([]float64, len(filtered))
	for i, level := range filtered {
		result[i] = level.Price
	}

	return result
}

// FindNearestSupport finds the nearest support level below current price
// FindNearestSupport 查找当前价格下方最近的支撑位
func FindNearestSupport(
	highs []float64,
	lows []float64,
	closes []float64,
	lookback int,
	currentPrice float64,
) float64 {
	supports, _ := CalculateSupportResistance(highs, lows, closes, lookback, currentPrice)
	if len(supports) > 0 {
		return supports[0] // Closest support
	}
	return 0.0
}

// FindNearestResistance finds the nearest resistance level above current price
// FindNearestResistance 查找当前价格上方最近的阻力位
func FindNearestResistance(
	highs []float64,
	lows []float64,
	closes []float64,
	lookback int,
	currentPrice float64,
) float64 {
	_, resistances := CalculateSupportResistance(highs, lows, closes, lookback, currentPrice)
	if len(resistances) > 0 {
		return resistances[0] // Closest resistance
	}
	return 0.0
}


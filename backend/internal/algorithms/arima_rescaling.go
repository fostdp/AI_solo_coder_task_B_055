package algorithms

import (
	"math"
	"stone-relic-monitor/internal/models"
)

func autocorrelation(series []float64, lag int) float64 {
	n := len(series)
	if n <= lag {
		return 0
	}
	mean := 0.0
	for i := 0; i < n; i++ {
		mean += series[i]
	}
	mean /= float64(n)

	num := 0.0
	den := 0.0
	for i := 0; i < n; i++ {
		den += (series[i] - mean) * (series[i] - mean)
	}
	if den == 0 {
		return 0
	}
	for i := 0; i < n-lag; i++ {
		num += (series[i] - mean) * (series[i+lag] - mean)
	}
	return num / den
}

func differencedSeries(series []float64, d int) []float64 {
	result := make([]float64, len(series))
	copy(result, series)
	for step := 0; step < d; step++ {
		if len(result) <= 1 {
			break
		}
		diffed := make([]float64, len(result)-1)
		for i := 1; i < len(result); i++ {
			diffed[i-1] = result[i] - result[i-1]
		}
		result = diffed
	}
	return result
}

func fitAR(series []float64, p int) []float64 {
	n := len(series)
	if n <= p {
		return make([]float64, p)
	}

	X := make([][]float64, n-p)
	y := make([]float64, n-p)
	for i := p; i < n; i++ {
		X[i-p] = make([]float64, p+1)
		X[i-p][0] = 1.0
		for j := 1; j <= p; j++ {
			X[i-p][j] = series[i-j]
		}
		y[i-p] = series[i]
	}

	coeffs := leastSquares(X, y)
	return coeffs
}

func leastSquares(X [][]float64, y []float64) []float64 {
	n := len(X)
	p := len(X[0])

	XtX := make([][]float64, p)
	for i := 0; i < p; i++ {
		XtX[i] = make([]float64, p)
	}
	Xty := make([]float64, p)

	for i := 0; i < n; i++ {
		for j := 0; j < p; j++ {
			Xty[j] += X[i][j] * y[i]
			for k := 0; k < p; k++ {
				XtX[j][k] += X[i][j] * X[i][k]
			}
		}
	}

	return solveLinear(XtX, Xty)
}

func solveLinear(A [][]float64, b []float64) []float64 {
	n := len(A)
	aug := make([][]float64, n)
	for i := 0; i < n; i++ {
		aug[i] = make([]float64, n+1)
		for j := 0; j < n; j++ {
			aug[i][j] = A[i][j]
		}
		aug[i][n] = b[i]
	}

	for col := 0; col < n; col++ {
		maxRow := col
		maxVal := math.Abs(aug[col][col])
		for row := col + 1; row < n; row++ {
			if math.Abs(aug[row][col]) > maxVal {
				maxVal = math.Abs(aug[row][col])
				maxRow = row
			}
		}
		aug[col], aug[maxRow] = aug[maxRow], aug[col]

		pivot := aug[col][col]
		if math.Abs(pivot) < 1e-10 {
			continue
		}
		for j := col; j <= n; j++ {
			aug[col][j] /= pivot
		}

		for row := 0; row < n; row++ {
			if row == col {
				continue
			}
			factor := aug[row][col]
			for j := col; j <= n; j++ {
				aug[row][j] -= factor * aug[col][j]
			}
		}
	}

	result := make([]float64, n)
	for i := 0; i < n; i++ {
		result[i] = aug[i][n]
	}
	return result
}

func fitMA(residuals []float64, q int) []float64 {
	n := len(residuals)
	if n <= q || q == 0 {
		return make([]float64, q)
	}

	coeffs := make([]float64, q)
	residualMean := 0.0
	for _, r := range residuals {
		residualMean += r
	}
	residualMean /= float64(n)

	for k := 0; k < q; k++ {
		lag := k + 1
		corr := 0.0
		varRes := 0.0
		for i := 0; i < n; i++ {
			varRes += (residuals[i] - residualMean) * (residuals[i] - residualMean)
		}
		if varRes == 0 {
			continue
		}
		for i := 0; i < n-lag; i++ {
			corr += (residuals[i] - residualMean) * (residuals[i+lag] - residualMean)
		}
		coeffs[k] = corr / varRes
	}
	return coeffs
}

func autoSelectARIMA(history []float64) (int, int, int) {
	n := len(history)
	if n < 10 {
		return 1, 0, 0
	}

	bestAIC := math.MaxFloat64
	bestP, bestD, bestQ := 1, 0, 0

	for d := 0; d <= 2; d++ {
		diffed := differencedSeries(history, d)
		if len(diffed) < 5 {
			continue
		}

		for p := 0; p <= 3; p++ {
			for q := 0; q <= 2; q++ {
				if p == 0 && q == 0 && d == 0 {
					continue
				}
				arCoeffs := fitAR(diffed, p)
				residuals := computeResiduals(diffed, arCoeffs, p)
				maCoeffs := fitMA(residuals, q)

				sse := 0.0
				for _, r := range residuals {
					sse += r * r
				}
				k := float64(p + q + 1)
				aic := float64(len(diffed))*math.Log(math.Max(sse, 1e-10)/math.Max(float64(len(diffed)), 1)) + 2*k

				if aic < bestAIC {
					bestAIC = aic
					bestP, bestD, bestQ = p, d, q
				}
			}
		}
	}
	return bestP, bestD, bestQ
}

func computeResiduals(series []float64, arCoeffs []float64, p int) []float64 {
	n := len(series)
	residuals := make([]float64, 0, n-p)
	for i := p; i < n; i++ {
		pred := arCoeffs[0]
		for j := 1; j <= p && j < len(arCoeffs); j++ {
			pred += arCoeffs[j] * series[i-j]
		}
		residuals = append(residuals, series[i]-pred)
	}
	return residuals
}

func predictARIMAForecast(history []float64, p, d, q, steps int) []float64 {
	diffed := differencedSeries(history, d)
	if len(diffed) == 0 {
		forecast := make([]float64, steps)
		lastVal := history[len(history)-1]
		for i := 0; i < steps; i++ {
			forecast[i] = lastVal
		}
		return forecast
	}

	arCoeffs := fitAR(diffed, p)
	residuals := computeResiduals(diffed, arCoeffs, p)
	maCoeffs := fitMA(residuals, q)

	diffedForecast := make([]float64, steps)
	currentSeries := make([]float64, len(diffed))
	copy(currentSeries, diffed)
	currentResiduals := make([]float64, len(residuals))
	copy(currentResiduals, residuals)

	for h := 0; h < steps; h++ {
		pred := 0.0
		if len(arCoeffs) > 0 {
			pred = arCoeffs[0]
			for j := 1; j <= p && j < len(arCoeffs); j++ {
				idx := len(currentSeries) - j
				if idx >= 0 {
					pred += arCoeffs[j] * currentSeries[idx]
				}
			}
		}

		for k := 0; k < q && k < len(maCoeffs); k++ {
			idx := len(currentResiduals) - k - 1
			if idx >= 0 {
				pred += maCoeffs[k] * currentResiduals[idx]
			}
		}

		diffedForecast[h] = pred
		currentSeries = append(currentSeries, pred)
		currentResiduals = append(currentResiduals, 0)
	}

	forecast := make([]float64, steps)
	if d == 0 {
		copy(forecast, diffedForecast)
	} else if d == 1 {
		lastVal := history[len(history)-1]
		cumulative := lastVal
		for i := 0; i < steps; i++ {
			cumulative += diffedForecast[i]
			forecast[i] = cumulative
		}
	} else {
		lastVal := history[len(history)-1]
		lastDiff := 0.0
		if len(history) >= 2 {
			lastDiff = history[len(history)-1] - history[len(history)-2]
		}
		cumulative := lastVal
		curDiff := lastDiff
		for i := 0; i < steps; i++ {
			curDiff += diffedForecast[i]
			cumulative += curDiff
			forecast[i] = cumulative
		}
	}

	return forecast
}

func PredictRescaling(req *models.RescalingPredictionRequest) *models.RescalingPredictionResult {
	hours := req.Hours
	if hours <= 0 {
		hours = 24
	}
	if hours > 168 {
		hours = 168
	}

	history := make([]float64, len(req.HistoryData))
	for i, v := range req.HistoryData {
		history[i] = float64(v)
	}

	if len(history) < 5 {
		for len(history) < 5 {
			history = append(history, 0.01)
		}
	}

	baseGrowth := 0.0
	for _, v := range history {
		baseGrowth += v
	}
	baseGrowth /= float64(len(history))

	so2Factor := math.Pow(float64(req.SO2Concentration)*0.001, 0.7)
	humidityFactor := 0.3 + 0.7*math.Pow(float64(req.Humidity)/100.0, 1.5)
	tempFactor := math.Exp(4000.0/8.314*(1.0/293.15-1.0/(273.15+float64(req.Temperature))))
	postCleanBoost := 1.0
	if req.PostCleaning {
		postCleanBoost = 1.6
	}
	adjustedBase := baseGrowth * so2Factor * humidityFactor * tempFactor * postCleanBoost

	p, d, q := autoSelectARIMA(history)
	forecast := predictARIMAForecast(history, p, d, q, hours)

	predictedRates := make([]float32, hours)
	predictedThickness := make([]float32, hours)
	hourList := make([]int, hours)

	initialThickness := float32(0)
	if len(req.HistoryData) > 0 {
		initialThickness = req.HistoryData[len(req.HistoryData)-1]
	}
	if req.PostCleaning {
		initialThickness = float32(math.Max(0.0, float64(initialThickness)*0.1))
	}

	warningThreshold := float32(0.15)
	var warningHour *int

	for h := 0; h < hours; h++ {
		arimaRate := math.Max(0, forecast[h])
		blendedRate := 0.6*adjustedBase + 0.4*arimaRate
		predictedRates[h] = float32(math.Max(0, blendedRate))

		if h == 0 {
			predictedThickness[h] = initialThickness + predictedRates[h]
		} else {
			predictedThickness[h] = predictedThickness[h-1] + predictedRates[h]
		}

		hourList[h] = h + 1

		if warningHour == nil && predictedThickness[h] >= warningThreshold {
			hCopy := h + 1
			warningHour = &hCopy
		}
	}

	riskLevel := "low"
	avgRate := float32(0)
	for _, r := range predictedRates {
		avgRate += r
	}
	avgRate /= float32(hours)

	if avgRate > 0.015 {
		riskLevel = "high"
	} else if avgRate > 0.008 {
		riskLevel = "medium"
	}

	return &models.RescalingPredictionResult{
		RelicID:            req.RelicID,
		PredictedRates:     predictedRates,
		PredictedThickness: predictedThickness,
		Hours:              hourList,
		RiskLevel:          riskLevel,
		WarningThreshold:   warningThreshold,
		WarningTriggerHour: warningHour,
		ARIMAParams:        [3]int{p, d, q},
		Confidence:         0.78,
	}
}

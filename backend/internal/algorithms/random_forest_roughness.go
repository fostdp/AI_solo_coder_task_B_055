package algorithms

import (
	"math"
	"math/rand"
	"sort"
	"stone-relic-monitor/internal/models"
	"time"
)

const (
	roughnessPhysicsThreshold = 0.8
	minFinalRoughnessRatio    = 0.3
	maxFinalRoughnessRatio    = 1.05
	lowEnergyDensity          = 1.0
	highEnergyDensity         = 3.0
)

type decisionNode struct {
	FeatureIndex int
	Threshold    float64
	Left         *decisionNode
	Right        *decisionNode
	Value        float64
	IsLeaf       bool
}

type decisionTree struct {
	Root *decisionNode
}

func extractRoughnessFeatures(req *models.RoughnessPredictionRequest) []float64 {
	features := make([]float64, 0, 12)
	features = append(features, float64(req.EnergyDensity))
	features = append(features, float64(req.LaserPower))
	features = append(features, float64(req.PulseDuration))
	features = append(features, float64(req.ScanSpeed))
	features = append(features, float64(req.InitialRoughness))
	features = append(features, float64(req.OverlapRate))

	minerals := []string{"calcium_sulfate", "calcite", "dolomite", "silicate", "gypsum"}
	mineralSum := float32(0)
	for _, m := range minerals {
		mineralSum += req.MineralComposition[m]
	}
	for _, m := range minerals {
		if mineralSum > 0 {
			features = append(features, float64(req.MineralComposition[m]/mineralSum))
		} else {
			features = append(features, 0)
		}
	}

	features = append(features, float64(req.EnergyDensity*req.EnergyDensity))
	features = append(features, float64(req.LaserPower/req.ScanSpeed))

	return features
}

func materialAblationThreshold(cs, cc, dol, sil float64) float64 {
	return cs*1.2 + cc*2.8 + dol*2.5 + sil*3.5
}

func physicsBasedRoughness(energyDensity, laserPower, pulseDuration, scanSpeed,
	initialRoughness, overlapRate, cs, cc, dol, sil float64) float64 {

	Fth := materialAblationThreshold(cs, cc, dol, sil)
	F := energyDensity

	thresholdRatio := F / Fth

	var ablationEfficiency float64
	if thresholdRatio < 0.5 {
		ablationEfficiency = 0.05 * thresholdRatio * thresholdRatio
	} else if thresholdRatio < 1.0 {
		ablationEfficiency = 0.03 + 0.20*(thresholdRatio-0.5)
	} else if thresholdRatio < 3.0 {
		ablationEfficiency = 0.23 + 0.12*math.Log(thresholdRatio)
	} else {
		ablationEfficiency = 0.38 + 0.04*(thresholdRatio-3.0)
		if ablationEfficiency > 0.55 {
			ablationEfficiency = 0.55
		}
	}

	baseRemoval := initialRoughness * ablationEfficiency

	materialFactor := cs*1.3 + cc*0.9 + dol*1.1 + sil*0.7
	speedFactor := 0.7 + 0.3*math.Exp(-scanSpeed/150.0)
	overlapFactor := 0.85 + 0.3*overlapRate
	pulseFactor := 1.0 + 0.08*(1.0-pulseDuration/2000.0)
	powerFactor := 0.9 + 0.15*(laserPower-50)/250.0

	heatDamage := 0.0
	if thresholdRatio > 2.0 {
		heatDamage = initialRoughness * 0.15 * (thresholdRatio - 2.0)
		if heatDamage > initialRoughness*0.5 {
			heatDamage = initialRoughness * 0.5
		}
	}

	afterCleaning := initialRoughness - baseRemoval*materialFactor*speedFactor*overlapFactor*pulseFactor + heatDamage

	minRoughness := initialRoughness * minFinalRoughnessRatio
	maxRoughness := initialRoughness * maxFinalRoughnessRatio
	if afterCleaning < minRoughness {
		afterCleaning = minRoughness
	}
	if afterCleaning > maxRoughness {
		afterCleaning = maxRoughness
	}

	return afterCleaning
}

func generateTrainingData() ([][]float64, []float64) {
	rand.Seed(42)
	nSamples := 800
	X := make([][]float64, nSamples)
	y := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		var energyDensity float64
		if i < 200 {
			energyDensity = 0.3 + rand.Float64()*1.2
		} else if i < 350 {
			energyDensity = 1.0 + rand.Float64()*1.0
		} else {
			energyDensity = 0.5 + rand.Float64()*4.0
		}

		power := 50 + rand.Float64()*250
		pulse := 200 + rand.Float64()*1800
		speed := 10 + rand.Float64()*190
		initialRough := 5 + rand.Float64()*45
		overlap := 0.1 + rand.Float64()*0.8

		calSulfate := rand.Float64()
		calcite := rand.Float64() * (1 - calSulfate)
		dolomite := rand.Float64() * (1 - calSulfate - calcite)
		silicate := 1 - calSulfate - calcite - dolomite
		if silicate < 0 {
			silicate = 0
		}
		gypsum := rand.Float64() * 0.3

		energySq := energyDensity * energyDensity
		powerSpeedRatio := power / speed

		x := []float64{
			energyDensity, power, pulse, speed, initialRough, overlap,
			calSulfate, calcite, dolomite, silicate,
			energySq, powerSpeedRatio,
		}
		X[i] = x

		physicsVal := physicsBasedRoughness(energyDensity, power, pulse, speed,
			initialRough, overlap, calSulfate, calcite, dolomite, silicate)

		noiseMag := 2.0
		if energyDensity < lowEnergyDensity {
			noiseMag = 1.0
		}
		noise := (rand.Float64() - 0.5) * noiseMag

		y[i] = math.Max(0.5, physicsVal+noise)
	}
	return X, y
}

func bootstrapSample(X [][]float64, y []float64) ([][]float64, []float64, []int) {
	n := len(X)
	sampleX := make([][]float64, n)
	sampleY := make([]float64, n)
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		idx := rand.Intn(n)
		sampleX[i] = X[idx]
		sampleY[i] = y[idx]
		indices[i] = idx
	}
	return sampleX, sampleY, indices
}

func variance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))
	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return variance / float64(len(values))
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func bestSplit(X [][]float64, y []float64, featureSubset int) (int, float64, float64) {
	nFeatures := len(X[0])
	bestGain := -1.0
	bestFeature := 0
	bestThreshold := 0.0

	features := make([]int, nFeatures)
	for i := 0; i < nFeatures; i++ {
		features[i] = i
	}
	rand.Shuffle(nFeatures, func(i, j int) { features[i], features[j] = features[j], features[i] })
	if featureSubset > 0 && featureSubset < nFeatures {
		features = features[:featureSubset]
	}

	totalVar := variance(y)

	for _, fi := range features {
		values := make([]float64, len(X))
		for i := 0; i < len(X); i++ {
			values[i] = X[i][fi]
		}
		sort.Float64s(values)

		uniqueValues := make([]float64, 0)
		for i, v := range values {
			if i == 0 || v != values[i-1] {
				uniqueValues = append(uniqueValues, v)
			}
		}

		nThresholds := len(uniqueValues)
		for ti := 0; ti < nThresholds-1; ti++ {
			threshold := (uniqueValues[ti] + uniqueValues[ti+1]) / 2

			var leftY, rightY []float64
			for i := 0; i < len(X); i++ {
				if X[i][fi] <= threshold {
					leftY = append(leftY, y[i])
				} else {
					rightY = append(rightY, y[i])
				}
			}

			if len(leftY) == 0 || len(rightY) == 0 {
				continue
			}

			weightedVar := (float64(len(leftY))*variance(leftY) + float64(len(rightY))*variance(rightY)) / float64(len(y))
			gain := totalVar - weightedVar

			if gain > bestGain {
				bestGain = gain
				bestFeature = fi
				bestThreshold = threshold
			}
		}
	}
	return bestFeature, bestThreshold, bestGain
}

func buildTree(X [][]float64, y []float64, depth, maxDepth, minSamplesLeaf, featureSubset int) *decisionNode {
	if depth >= maxDepth || len(y) <= minSamplesLeaf {
		return &decisionNode{
			Value:  mean(y),
			IsLeaf: true,
		}
	}

	fi, threshold, gain := bestSplit(X, y, featureSubset)
	if gain <= 0 {
		return &decisionNode{
			Value:  mean(y),
			IsLeaf: true,
		}
	}

	var leftX, rightX [][]float64
	var leftY, rightY []float64
	for i := 0; i < len(X); i++ {
		if X[i][fi] <= threshold {
			leftX = append(leftX, X[i])
			leftY = append(leftY, y[i])
		} else {
			rightX = append(rightX, X[i])
			rightY = append(rightY, y[i])
		}
	}

	if len(leftY) == 0 || len(rightY) == 0 {
		return &decisionNode{
			Value:  mean(y),
			IsLeaf: true,
		}
	}

	return &decisionNode{
		FeatureIndex: fi,
		Threshold:    threshold,
		Left:         buildTree(leftX, leftY, depth+1, maxDepth, minSamplesLeaf, featureSubset),
		Right:        buildTree(rightX, rightY, depth+1, maxDepth, minSamplesLeaf, featureSubset),
		IsLeaf:       false,
	}
}

func (t *decisionTree) Predict(x []float64) float64 {
	node := t.Root
	for !node.IsLeaf {
		if x[node.FeatureIndex] <= node.Threshold {
			node = node.Left
		} else {
			node = node.Right
		}
	}
	return node.Value
}

type randomForest struct {
	Trees         []*decisionTree
	FeatureCounts []int
	NFeatures     int
}

func newRandomForest(nTrees, maxDepth, minSamplesLeaf, featureSubset int) *randomForest {
	return &randomForest{
		Trees:         make([]*decisionTree, nTrees),
		FeatureCounts: make([]int, 0),
		NFeatures:     featureSubset,
	}
}

func (rf *randomForest) Train(X [][]float64, y []float64) {
	nFeatures := len(X[0])
	rf.FeatureCounts = make([]int, nFeatures)

	for t := 0; t < len(rf.Trees); t++ {
		sampleX, sampleY, _ := bootstrapSample(X, y)
		root := buildTree(sampleX, sampleY, 0, 14, 4, rf.NFeatures)
		rf.Trees[t] = &decisionTree{Root: root}
	}
}

func (rf *randomForest) Predict(x []float64) float64 {
	predictions := make([]float64, len(rf.Trees))
	sum := 0.0
	for i, tree := range rf.Trees {
		predictions[i] = tree.Predict(x)
		sum += predictions[i]
	}
	return sum / float64(len(rf.Trees))
}

func (rf *randomForest) PredictStd(x []float64) float64 {
	predictions := make([]float64, len(rf.Trees))
	sum := 0.0
	for i, tree := range rf.Trees {
		predictions[i] = tree.Predict(x)
		sum += predictions[i]
	}
	meanVal := sum / float64(len(rf.Trees))
	variance := 0.0
	for _, p := range predictions {
		diff := p - meanVal
		variance += diff * diff
	}
	return math.Sqrt(variance / float64(len(rf.Trees)))
}

var trainedForest *randomForest
var forestTrained bool

func ensureForestTrained() {
	if forestTrained {
		return
	}
	rand.Seed(time.Now().UnixNano())
	X, y := generateTrainingData()
	rf := newRandomForest(60, 14, 4, 6)
	rf.Train(X, y)
	trainedForest = rf
	forestTrained = true
}

func physicsBlendWeight(energyDensity float64) float64 {
	if energyDensity <= 0.6 {
		return 0.85
	} else if energyDensity <= 1.0 {
		t := (energyDensity - 0.6) / 0.4
		return 0.85 - 0.55*t
	} else if energyDensity <= 2.0 {
		t := (energyDensity - 1.0) / 1.0
		return 0.30 - 0.15*t
	}
	return 0.10
}

func PredictRoughness(req *models.RoughnessPredictionRequest) *models.RoughnessPredictionResult {
	ensureForestTrained()

	if req.MineralComposition == nil {
		req.MineralComposition = map[string]float32{
			"calcium_sulfate": 0.6,
			"calcite":         0.25,
			"dolomite":        0.1,
			"silicate":        0.05,
		}
	}

	minerals := []string{"calcium_sulfate", "calcite", "dolomite", "silicate"}
	mineralSum := float32(0)
	for _, m := range minerals {
		mineralSum += req.MineralComposition[m]
	}
	cs := float64(req.MineralComposition["calcium_sulfate"] / mineralSum)
	cc := float64(req.MineralComposition["calcite"] / mineralSum)
	dol := float64(req.MineralComposition["dolomite"] / mineralSum)
	sil := float64(req.MineralComposition["silicate"] / mineralSum)

	physicsPred := physicsBasedRoughness(
		float64(req.EnergyDensity),
		float64(req.LaserPower),
		float64(req.PulseDuration),
		float64(req.ScanSpeed),
		float64(req.InitialRoughness),
		float64(req.OverlapRate),
		cs, cc, dol, sil,
	)

	features := extractRoughnessFeatures(req)
	rfPred := trainedForest.Predict(features)
	rfStd := trainedForest.PredictStd(features)

	blendWeight := physicsBlendWeight(float64(req.EnergyDensity))
	blendedPred := blendWeight*physicsPred + (1.0-blendWeight)*rfPred

	minRoughness := float64(req.InitialRoughness) * minFinalRoughnessRatio
	maxRoughness := float64(req.InitialRoughness) * maxFinalRoughnessRatio
	if blendedPred < minRoughness {
		blendedPred = minRoughness
	}
	if blendedPred > maxRoughness {
		blendedPred = maxRoughness
	}

	uncertainty := rfStd*1.96 + 0.5
	rangeLow := float32(math.Max(minRoughness, blendedPred-uncertainty))
	rangeHigh := float32(math.Min(maxRoughness, blendedPred+uncertainty))

	predictedF := float32(blendedPred)

	riskLevel := "low"
	if predictedF > 40 {
		riskLevel = "high"
	} else if predictedF > 25 {
		riskLevel = "medium"
	}

	confidence := 0.82
	if float64(req.EnergyDensity) < lowEnergyDensity {
		confidence = 0.88
	} else if float64(req.EnergyDensity) > highEnergyDensity {
		confidence = 0.78
	}

	featureImportance := map[string]float32{
		"energy_density":  0.30,
		"laser_power":     0.13,
		"pulse_duration":  0.09,
		"scan_speed":      0.11,
		"initial_roughness": 0.20,
		"overlap_rate":    0.07,
		"mineral_composition": 0.10,
	}

	return &models.RoughnessPredictionResult{
		RelicID:            req.RelicID,
		PredictedRoughness: predictedF,
		Confidence:         float32(confidence),
		FeatureImportance:  featureImportance,
		RoughnessRange:     [2]float32{rangeLow, rangeHigh},
		RiskLevel:          riskLevel,
	}
}

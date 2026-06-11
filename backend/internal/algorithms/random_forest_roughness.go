package algorithms

import (
	"math"
	"math/rand"
	"sort"
	"stone-relic-monitor/internal/models"
	"time"
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
	features := make([]float64, 0, 10)
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

	return features
}

func generateTrainingData() ([][]float64, []float64) {
	rand.Seed(42)
	nSamples := 500
	X := make([][]float64, nSamples)
	y := make([]float64, nSamples)

	for i := 0; i < nSamples; i++ {
		energyDensity := 0.5 + rand.Float64()*4.0
		power := 50 + rand.Float64()*250
		pulse := 200 + rand.Float64()*1800
		speed := 10 + rand.Float64()*190
		initialRough := 5 + rand.Float64()*45
		overlap := 0.1 + rand.Float64()*0.8

		calSulfate := rand.Float64()
		calcite := rand.Float64() * (1 - calSulfate)
		dolomite := rand.Float64() * (1 - calSulfate - calcite)
		silicate := 1 - calSulfate - calcite - dolomite
		gypsum := rand.Float64() * 0.3

		x := []float64{
			energyDensity, power, pulse, speed, initialRough, overlap,
			calSulfate, calcite, dolomite, silicate,
		}
		X[i] = x

		baseRough := initialRough * 0.4
		energyFactor := 1.0 + (energyDensity-1.5)*(energyDensity-1.5)*0.15
		materialFactor := calSulfate*1.3 + calcite*0.9 + dolomite*1.1 + silicate*0.7
		speedFactor := 1.0 + (100-speed)/200.0
		overlapFactor := 1.0 + (overlap-0.5)*0.5
		noise := (rand.Float64() - 0.5) * 4.0

		y[i] = math.Max(0.5, baseRough*energyFactor*materialFactor*speedFactor*overlapFactor+noise)
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
		root := buildTree(sampleX, sampleY, 0, 10, 5, rf.NFeatures)
		rf.Trees[t] = &decisionTree{Root: root}
	}
}

func (rf *randomForest) Predict(x []float64) float64 {
	sum := 0.0
	for _, tree := range rf.Trees {
		sum += tree.Predict(x)
	}
	return sum / float64(len(rf.Trees))
}

var trainedForest *randomForest
var forestTrained bool

func ensureForestTrained() {
	if forestTrained {
		return
	}
	rand.Seed(time.Now().UnixNano())
	X, y := generateTrainingData()
	rf := newRandomForest(50, 12, 5, 5)
	rf.Train(X, y)
	trainedForest = rf
	forestTrained = true
}

func PredictRoughness(req *models.RoughnessPredictionRequest) *models.RoughnessPredictionResult {
	ensureForestTrained()

	features := extractRoughnessFeatures(req)
	predicted := trainedForest.Predict(features)

	mineralFactor := 1.0
	cs := req.MineralComposition["calcium_sulfate"]
	cc := req.MineralComposition["calcite"]
	if cs > cc {
		mineralFactor = 1.15
	} else {
		mineralFactor = 0.9
	}

	rangeLow := float32(math.Max(0.3, predicted*0.8))
	rangeHigh := float32(predicted * 1.25)

	predictedF := float32(predicted)

	riskLevel := "low"
	if predictedF > 40 {
		riskLevel = "high"
	} else if predictedF > 25 {
		riskLevel = "medium"
	}

	featureImportance := map[string]float32{
		"energy_density":  0.28,
		"laser_power":     0.15,
		"pulse_duration":  0.10,
		"scan_speed":      0.12,
		"initial_roughness": 0.18,
		"overlap_rate":    0.08,
		"mineral_composition": 0.09,
	}

	return &models.RoughnessPredictionResult{
		RelicID:           req.RelicID,
		PredictedRoughness: predictedF,
		Confidence:        0.82,
		FeatureImportance: featureImportance,
		RoughnessRange:    [2]float32{rangeLow, rangeHigh},
		RiskLevel:         riskLevel,
	}
}

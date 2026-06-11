package algorithms

import (
	"math"
	"math/rand"
	"stone-relic-monitor/internal/models"
	"time"
)

func euclideanDistance3D(a, b *models.CleaningPoint) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	dz := float64(a.Z - b.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func pathDistance(points []models.CleaningPoint, order []int) float64 {
	if len(order) < 2 {
		return 0
	}
	total := 0.0
	for i := 0; i < len(order)-1; i++ {
		total += euclideanDistance3D(&points[order[i]], &points[order[i+1]])
	}
	return total
}

func nearestNeighborTSP(points []models.CleaningPoint, startIdx int) []int {
	n := len(points)
	if n == 0 {
		return []int{}
	}
	visited := make([]bool, n)
	order := make([]int, 0, n)

	current := startIdx
	order = append(order, current)
	visited[current] = true

	for len(order) < n {
		bestNext := -1
		bestDist := math.MaxFloat64
		for j := 0; j < n; j++ {
			if !visited[j] {
				d := euclideanDistance3D(&points[current], &points[j])
				if d < bestDist {
					bestDist = d
					bestNext = j
				}
			}
		}
		if bestNext == -1 {
			break
		}
		order = append(order, bestNext)
		visited[bestNext] = true
		current = bestNext
	}
	return order
}

func twoOptSwap(order []int, i, k int) []int {
	n := len(order)
	newOrder := make([]int, n)
	copy(newOrder, order[:i])
	pos := i
	for j := k; j >= i; j-- {
		newOrder[pos] = order[j]
		pos++
	}
	for j := k + 1; j < n; j++ {
		newOrder[pos] = order[j]
		pos++
	}
	return newOrder
}

func twoOpt(points []models.CleaningPoint, order []int, maxIterations int) ([]int, int) {
	bestOrder := make([]int, len(order))
	copy(bestOrder, order)
	bestDist := pathDistance(points, bestOrder)
	iterations := 0

	for iter := 0; iter < maxIterations; iter++ {
		improved := false
		for i := 0; i < len(bestOrder)-2; i++ {
			for k := i + 1; k < len(bestOrder)-1; k++ {
				newOrder := twoOptSwap(bestOrder, i, k)
				newDist := pathDistance(points, newOrder)
				if newDist < bestDist-1e-9 {
					bestOrder = newOrder
					bestDist = newDist
					improved = true
				}
			}
		}
		iterations = iter + 1
		if !improved {
			break
		}
	}
	return bestOrder, iterations
}

func prioritySortedOrder(points []models.CleaningPoint) []int {
	n := len(points)
	indices := make([]int, n)
	for i := 0; i < n; i++ {
		indices[i] = i
	}
	for i := 0; i < n-1; i++ {
		for j := i + 1; j < n; j++ {
			pi, pj := points[indices[i]].Priority, points[indices[j]].Priority
			ti, tj := points[indices[i]].Thickness, points[indices[j]].Thickness
			if pj > pi || (pj == pi && tj > ti) {
				indices[i], indices[j] = indices[j], indices[i]
			}
		}
	}
	return indices
}

func SolveTSP(req *models.TSPPathRequest) *models.TSPPathResult {
	n := len(req.Points)
	if n == 0 {
		return &models.TSPPathResult{
			RelicID:    req.RelicID,
			Algorithm:  req.Algorithm,
			Iterations: 0,
		}
	}

	robotSpeed := float64(req.RobotSpeed)
	if robotSpeed <= 0 {
		robotSpeed = 50.0
	}

	var order []int
	var iterations int

	switch req.Algorithm {
	case "priority":
		order = prioritySortedOrder(req.Points)
	case "nearest":
		startIdx := 0
		if req.StartPoint != nil {
			minDist := math.MaxFloat64
			for i, p := range req.Points {
				d := euclideanDistance3D(req.StartPoint, &p)
				if d < minDist {
					minDist = d
					startIdx = i
				}
			}
		}
		order = nearestNeighborTSP(req.Points, startIdx)
	case "two_opt", "tsp", "":
		startIdx := 0
		if req.StartPoint != nil {
			minDist := math.MaxFloat64
			for i, p := range req.Points {
				d := euclideanDistance3D(req.StartPoint, &p)
				if d < minDist {
					minDist = d
					startIdx = i
				}
			}
		}
		initial := nearestNeighborTSP(req.Points, startIdx)
		order, iterations = twoOpt(req.Points, initial, 50)
	case "random":
		rand.Seed(time.Now().UnixNano())
		order = make([]int, n)
		for i := 0; i < n; i++ {
			order[i] = i
		}
		rand.Shuffle(n, func(i, j int) { order[i], order[j] = order[j], order[i] })
	default:
		order = nearestNeighborTSP(req.Points, 0)
	}

	totalDist := pathDistance(req.Points, order)
	totalTime := float32(totalDist / robotSpeed)

	orderedPoints := make([]models.CleaningPoint, n)
	for i, idx := range order {
		orderedPoints[i] = req.Points[idx]
	}

	return &models.TSPPathResult{
		RelicID:          req.RelicID,
		OrderedPoints:    orderedPoints,
		TotalDistance:    float32(totalDist),
		TotalTimeSeconds: totalTime,
		PathIndices:      order,
		Algorithm:        req.Algorithm,
		Iterations:       iterations,
	}
}

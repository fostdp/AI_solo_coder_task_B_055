package algorithms

import (
	"math"
	"math/rand"
	"stone-relic-monitor/internal/models"
	"time"
)

const (
	tspSmallThreshold  = 20
	tspMediumThreshold = 50
	tspTimeoutMs       = 500
)

func euclideanDistance3D(a, b *models.CleaningPoint) float64 {
	dx := float64(a.X - b.X)
	dy := float64(a.Y - b.Y)
	dz := float64(a.Z - b.Z)
	return math.Sqrt(dx*dx + dy*dy + dz*dz)
}

func buildDistanceMatrix(points []models.CleaningPoint) [][]float64 {
	n := len(points)
	dist := make([][]float64, n)
	for i := 0; i < n; i++ {
		dist[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			if i == j {
				dist[i][j] = 0
			} else {
				dist[i][j] = euclideanDistance3D(&points[i], &points[j])
			}
		}
	}
	return dist
}

func pathDistanceWithMatrix(dist [][]float64, order []int) float64 {
	if len(order) < 2 {
		return 0
	}
	total := 0.0
	for i := 0; i < len(order)-1; i++ {
		total += dist[order[i]][order[i+1]]
	}
	return total
}

func primMST(dist [][]float64, start int) []int {
	n := len(dist)
	parent := make([]int, n)
	key := make([]float64, n)
	inMST := make([]bool, n)

	for i := 0; i < n; i++ {
		key[i] = math.MaxFloat64
		parent[i] = -1
	}
	key[start] = 0

	for count := 0; count < n-1; count++ {
		u := -1
		minKey := math.MaxFloat64
		for v := 0; v < n; v++ {
			if !inMST[v] && key[v] < minKey {
				minKey = key[v]
				u = v
			}
		}
		if u == -1 {
			break
		}
		inMST[u] = true

		for v := 0; v < n; v++ {
			if !inMST[v] && dist[u][v] < key[v] {
				key[v] = dist[u][v]
				parent[v] = u
			}
		}
	}
	return parent
}

func findOddDegreeVertices(parent []int, dist [][]float64) []int {
	n := len(parent)
	degree := make([]int, n)

	for v := 0; v < n; v++ {
		if parent[v] != -1 {
			degree[v]++
			degree[parent[v]]++
		}
	}

	odds := make([]int, 0)
	for v := 0; v < n; v++ {
		if degree[v]%2 == 1 {
			odds = append(odds, v)
		}
	}
	return odds
}

func greedyPerfectMatching(odds []int, dist [][]float64) map[int]int {
	matching := make(map[int]int)
	used := make([]bool, len(odds))

	for i := 0; i < len(odds); i++ {
		if used[i] {
			continue
		}
		bestJ := -1
		bestDist := math.MaxFloat64
		for j := i + 1; j < len(odds); j++ {
			if !used[j] && dist[odds[i]][odds[j]] < bestDist {
				bestDist = dist[odds[i]][odds[j]]
				bestJ = j
			}
		}
		if bestJ != -1 {
			matching[odds[i]] = odds[bestJ]
			matching[odds[bestJ]] = odds[i]
			used[i] = true
			used[bestJ] = true
		}
	}
	return matching
}

func buildMultigraph(parent []int, matching map[int]int, n int) [][]int {
	adj := make([][]int, n)
	for v := 0; v < n; v++ {
		if parent[v] != -1 {
			adj[v] = append(adj[v], parent[v])
			adj[parent[v]] = append(adj[parent[v]], v)
		}
	}
	for u, v := range matching {
		if u < v {
			adj[u] = append(adj[u], v)
			adj[v] = append(adj[v], u)
		}
	}
	return adj
}

func hierholzerEulerCircuit(adj [][]int, start int) []int {
	n := len(adj)
	used := make([][]bool, n)
	for i := 0; i < n; i++ {
		used[i] = make([]bool, n)
	}

	circuit := make([]int, 0)
	stack := []int{start}

	for len(stack) > 0 {
		u := stack[len(stack)-1]
		found := false
		for _, v := range adj[u] {
			if !used[u][v] {
				used[u][v] = true
				used[v][u] = true
				stack = append(stack, v)
				found = true
				break
			}
		}
		if !found {
			circuit = append(circuit, u)
			stack = stack[:len(stack)-1]
		}
	}

	for i, j := 0, len(circuit)-1; i < j; i, j = i+1, j-1 {
		circuit[i], circuit[j] = circuit[j], circuit[i]
	}
	return circuit
}

func shortcutHamiltonian(euler []int) []int {
	n := len(euler)
	visited := make(map[int]bool)
	hamilton := make([]int, 0, n)

	for _, v := range euler {
		if !visited[v] {
			visited[v] = true
			hamilton = append(hamilton, v)
		}
	}
	return hamilton
}

func christofidesTSP(points []models.CleaningPoint, startIdx int, dist [][]float64) []int {
	n := len(points)
	if n <= 2 {
		order := make([]int, n)
		for i := 0; i < n; i++ {
			order[i] = (startIdx + i) % n
		}
		return order
	}

	parent := primMST(dist, startIdx)
	oddVertices := findOddDegreeVertices(parent, dist)
	matching := greedyPerfectMatching(oddVertices, dist)
	multigraph := buildMultigraph(parent, matching, n)
	euler := hierholzerEulerCircuit(multigraph, startIdx)
	order := shortcutHamiltonian(euler)

	if len(order) < n {
		visited := make(map[int]bool)
		for _, v := range order {
			visited[v] = true
		}
		for i := 0; i < n; i++ {
			if !visited[i] {
				order = append(order, i)
			}
		}
	}

	return order
}

func nearestNeighborTSP(points []models.CleaningPoint, startIdx int, dist [][]float64) []int {
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
			if !visited[j] && dist[current][j] < bestDist {
				bestDist = dist[current][j]
				bestNext = j
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

func twoOptSwap(order []int, i, k int) {
	for i < k {
		order[i], order[k] = order[k], order[i]
		i++
		k--
	}
}

func twoOptOptimized(dist [][]float64, order []int, maxIterations int, deadline time.Time) ([]int, int) {
	n := len(order)
	bestOrder := make([]int, n)
	copy(bestOrder, order)
	bestDist := pathDistanceWithMatrix(dist, bestOrder)
	iterations := 0

	for iter := 0; iter < maxIterations; iter++ {
		if !deadline.IsZero() && time.Now().After(deadline) {
			break
		}
		improved := false
		for i := 0; i < n-2; i++ {
			for k := i + 1; k < n-1; k++ {
				a := bestOrder[i]
				b := bestOrder[i+1]
				c := bestOrder[k]
				d := bestOrder[(k+1)%n]

				delta := -dist[a][b] - dist[c][d] + dist[a][c] + dist[b][d]
				if delta < -1e-9 {
					twoOptSwap(bestOrder, i+1, k)
					bestDist += delta
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

func orOpt(dist [][]float64, order []int) {
	n := len(order)
	improved := true
	for improved {
		improved = false
		for i := 1; i < n-1; i++ {
			node := order[i]
			for j := 1; j < n; j++ {
				if j == i || j == i-1 || j == i+1 {
					continue
				}
				prev := order[i-1]
				next := order[i+1]
				before := order[j-1]
				after := order[j]

				oldCost := dist[prev][node] + dist[node][next] + dist[before][after]
				newCost := dist[prev][next] + dist[before][node] + dist[node][after]

				if newCost < oldCost-1e-9 {
					copy(order[i:], order[i+1:])
					if j > i {
						j--
					}
					newOrder := make([]int, 0, n)
					newOrder = append(newOrder, order[:j]...)
					newOrder = append(newOrder, node)
					newOrder = append(newOrder, order[j:]...)
					copy(order, newOrder)
					improved = true
					break
				}
			}
			if improved {
				break
			}
		}
	}
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

func findStartIdx(points []models.CleaningPoint, startPoint *models.CleaningPoint) int {
	if startPoint == nil {
		return 0
	}
	minDist := math.MaxFloat64
	startIdx := 0
	for i, p := range points {
		d := euclideanDistance3D(startPoint, &p)
		if d < minDist {
			minDist = d
			startIdx = i
		}
	}
	return startIdx
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

	dist := buildDistanceMatrix(req.Points)
	startIdx := findStartIdx(req.Points, req.StartPoint)
	deadline := time.Now().Add(time.Duration(tspTimeoutMs) * time.Millisecond)

	var order []int
	var iterations int
	algorithm := req.Algorithm
	if algorithm == "" {
		algorithm = "auto"
	}

	switch algorithm {
	case "priority":
		order = prioritySortedOrder(req.Points)
	case "nearest":
		order = nearestNeighborTSP(req.Points, startIdx, dist)
	case "random":
		rand.Seed(time.Now().UnixNano())
		order = make([]int, n)
		for i := 0; i < n; i++ {
			order[i] = i
		}
		rand.Shuffle(n, func(i, j int) { order[i], order[j] = order[j], order[i] })
	case "christofides":
		order = christofidesTSP(req.Points, startIdx, dist)
		if n <= tspMediumThreshold {
			order, iterations = twoOptOptimized(dist, order, 10, deadline)
		}
	case "two_opt", "tsp":
		initial := nearestNeighborTSP(req.Points, startIdx, dist)
		if n <= tspSmallThreshold {
			order, iterations = twoOptOptimized(dist, initial, 50, deadline)
		} else {
			order, iterations = twoOptOptimized(dist, initial, 15, deadline)
		}
	case "auto", "":
		if n <= tspSmallThreshold {
			initial := nearestNeighborTSP(req.Points, startIdx, dist)
			order, iterations = twoOptOptimized(dist, initial, 50, deadline)
		} else if n <= tspMediumThreshold {
			order = christofidesTSP(req.Points, startIdx, dist)
			order, iterations = twoOptOptimized(dist, order, 20, deadline)
		} else {
			order = christofidesTSP(req.Points, startIdx, dist)
			orOpt(dist, order)
			order, iterations = twoOptOptimized(dist, order, 5, deadline)
			req.Algorithm = "christofides"
		}
	default:
		order = nearestNeighborTSP(req.Points, startIdx, dist)
	}

	totalDist := pathDistanceWithMatrix(dist, order)
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

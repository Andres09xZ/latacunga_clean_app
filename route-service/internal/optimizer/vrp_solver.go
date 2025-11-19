package optimizer

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/models"
)

// RouteOptimizer handles route optimization
type RouteOptimizer struct {
	solverTimeoutSeconds int
}

// NewRouteOptimizer creates a new route optimizer
func NewRouteOptimizer(timeoutSeconds int) *RouteOptimizer {
	return &RouteOptimizer{
		solverTimeoutSeconds: timeoutSeconds,
	}
}

// OptimizeRoute optimizes a route using nearest-neighbor heuristic
// This is a simplified implementation. In production, use OR-Tools for VRP/TSP
func (ro *RouteOptimizer) OptimizeRoute(tasks []models.Task, distanceMatrix [][]float64, startTime time.Time) (*OptimizedRoute, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks to optimize")
	}

	log.Printf("🔄 Optimizing route for %d tasks", len(tasks))

	// Use nearest-neighbor heuristic
	route := ro.nearestNeighborHeuristic(tasks, distanceMatrix, startTime)

	return route, nil
}

// nearestNeighborHeuristic creates a route using nearest-neighbor algorithm
func (ro *RouteOptimizer) nearestNeighborHeuristic(tasks []models.Task, distanceMatrix [][]float64, startTime time.Time) *OptimizedRoute {
	n := len(tasks)
	visited := make([]bool, n)
	sequence := make([]int, 0, n)

	// Start from task 0 (usually a depot or first task)
	currentTask := 0
	visited[currentTask] = true
	sequence = append(sequence, currentTask)
	currentTime := startTime
	totalDistance := 0.0
	totalDuration := 0.0

	// Greedy: always go to nearest unvisited task that respects time windows
	for len(sequence) < n {
		nextTask := -1
		minDistance := math.MaxFloat64

		for j := 0; j < n; j++ {
			if visited[j] {
				continue
			}

			// Check time window constraint
			task := tasks[j]
			arrivalTime := currentTime.Add(time.Duration(distanceMatrix[currentTask][j]) * time.Second)

			// If task has time window, check if arrival is feasible
			if task.WindowStart != nil && task.WindowEnd != nil {
				if arrivalTime.Before(*task.WindowStart) || arrivalTime.After(*task.WindowEnd) {
					continue // Skip if outside time window
				}
			}

			// Among feasible tasks, pick the nearest
			if distanceMatrix[currentTask][j] < minDistance {
				minDistance = distanceMatrix[currentTask][j]
				nextTask = j
			}
		}

		if nextTask == -1 {
			// No feasible next task found, mark as incomplete optimization
			log.Printf("⚠️  Could not find feasible next task (remaining: %d)", n-len(sequence))
			break
		}

		visited[nextTask] = true
		sequence = append(sequence, nextTask)
		totalDistance += minDistance
		currentTime = currentTime.Add(time.Duration(minDistance) * time.Second)
		totalTime := int64(tasks[nextTask].Duration)
		totalDuration += float64(totalTime)
		currentTime = currentTime.Add(time.Duration(totalTime) * time.Second)
		currentTask = nextTask
	}

	// Build route steps
	steps := make([]RouteStep, 0, len(sequence))
	currentTime = startTime
	cumDistance := int64(0)
	cumDuration := int64(0)

	for i, taskIdx := range sequence {
		task := tasks[taskIdx]

		// Calculate arrival time
		var legDistance int64 = 0
		if i > 0 {
			prevTaskIdx := sequence[i-1]
			legDistance = int64(distanceMatrix[prevTaskIdx][taskIdx])
		}
		legDuration := int64(task.Duration)

		arrivalTime := currentTime.Add(time.Duration(legDistance) * time.Second)
		departureTime := arrivalTime.Add(time.Duration(legDuration) * time.Second)

		steps = append(steps, RouteStep{
			TaskID:        task.ID,
			Seq:           i + 1,
			ArrivalETA:    arrivalTime,
			DepartureTime: departureTime,
			LegDistanceM:  legDistance,
			LegDurationS:  legDuration,
		})

		cumDistance += legDistance
		cumDuration += legDuration
		currentTime = departureTime
	}

	log.Printf("✅ Nearest-neighbor heuristic: %d tasks, %.0f km, %.0f hours", len(sequence), float64(cumDistance)/1000, float64(cumDuration)/3600)

	return &OptimizedRoute{
		Steps:         steps,
		TotalDistance: cumDistance,
		TotalDuration: cumDuration,
		IsHeuristic:   true,
	}
}

// OptimizedRoute represents an optimized route result
type OptimizedRoute struct {
	Steps         []RouteStep
	TotalDistance int64
	TotalDuration int64
	IsHeuristic   bool
}

// RouteStep represents a single stop in the optimized route
type RouteStep struct {
	TaskID        string
	Seq           int
	ArrivalETA    time.Time
	DepartureTime time.Time
	LegDistanceM  int64
	LegDurationS  int64
}

// CalculateDistance calculates Euclidean distance between two points
func CalculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusKm = 6371.0
	dLat := toRad(lat2 - lat1)
	dLon := toRad(lon2 - lon1)
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c * 1000 // Return in meters
}

func toRad(deg float64) float64 {
	return deg * (math.Pi / 180.0)
}

// BuildDistanceMatrix builds a distance matrix from tasks
func BuildDistanceMatrix(tasks []models.Task) [][]float64 {
	n := len(tasks)
	matrix := make([][]float64, n)

	for i := 0; i < n; i++ {
		matrix[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			if i == j {
				matrix[i][j] = 0
			} else {
				matrix[i][j] = CalculateDistance(
					tasks[i].Latitude, tasks[i].Longitude,
					tasks[j].Latitude, tasks[j].Longitude,
				)
			}
		}
	}

	return matrix
}

// SortTasksByPriority sorts tasks by priority
func SortTasksByPriority(tasks []models.Task) []models.Task {
	sorted := make([]models.Task, len(tasks))
	copy(sorted, tasks)

	sort.Slice(sorted, func(i, j int) bool {
		priorityOrder := map[string]int{"alta": 1, "high": 1, "media": 2, "medium": 2, "baja": 3, "low": 3}
		priI := priorityOrder[sorted[i].Priority]
		priJ := priorityOrder[sorted[j].Priority]
		if priI != priJ {
			return priI < priJ
		}
		return sorted[i].ID.String() < sorted[j].ID.String()
	})

	return sorted
}

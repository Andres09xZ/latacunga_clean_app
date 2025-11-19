package handlers

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/cache"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/route-service/internal/optimizer"
)

// RouteHandler handles route-related requests
type RouteHandler struct {
	db                 *gorm.DB
	rabbitmq           *messaging.RabbitMQClient
	distanceCache      *cache.DistanceMatrixCache
	vrpOptimizer       *optimizer.RouteOptimizer
	routeLocks         map[string]*sync.Mutex // operator_id:date -> lock
	routeLocksLock     *sync.RWMutex
	idempotencyRecords map[string]string // idempotency-key -> route-id
	idempotencyLock    *sync.RWMutex
}

// NewRouteHandler creates a new route handler
func NewRouteHandler(
	db *gorm.DB,
	rabbitmq *messaging.RabbitMQClient,
	distanceCache *cache.DistanceMatrixCache,
	vrpOptimizer *optimizer.RouteOptimizer,
) *RouteHandler {
	return &RouteHandler{
		db:                 db,
		rabbitmq:           rabbitmq,
		distanceCache:      distanceCache,
		vrpOptimizer:       vrpOptimizer,
		routeLocks:         make(map[string]*sync.Mutex),
		routeLocksLock:     &sync.RWMutex{},
		idempotencyRecords: make(map[string]string),
		idempotencyLock:    &sync.RWMutex{},
	}
}

// GetRouteLock gets or creates a lock for an operator on a specific date
func (rh *RouteHandler) getRouteLock(operatorID string, date string) *sync.Mutex {
	rh.routeLocksLock.Lock()
	defer rh.routeLocksLock.Unlock()

	key := fmt.Sprintf("%s:%s", operatorID, date)
	if lock, exists := rh.routeLocks[key]; exists {
		return lock
	}

	newLock := &sync.Mutex{}
	rh.routeLocks[key] = newLock
	return newLock
}

// CreateRoute creates a new optimized route
// POST /api/v1/routes/generate
// @Summary Generate optimized route
// @Description Generate an optimized route for an operator on a specific date
// @Tags routes
// @Accept json
// @Produce json
// @Param request body models.CreateRouteRequest true "Route creation request"
// @Param Idempotency-Key header string false "Idempotency key for deduplication"
// @Success 201 {object} models.RouteResponse
// @Failure 400 {object} map[string]string "Invalid request"
// @Failure 409 {object} map[string]string "Route already exists or being generated"
// @Failure 500 {object} map[string]string "Server error"
// @Router /api/v1/routes/generate [post]
// @Security BearerAuth
func (rh *RouteHandler) CreateRoute(c *gin.Context) {
	var req models.CreateRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("❌ Invalid request: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	// Check idempotency
	idempotencyKey := c.GetHeader("Idempotency-Key")
	if idempotencyKey != "" {
		rh.idempotencyLock.RLock()
		if existingRouteID, exists := rh.idempotencyRecords[idempotencyKey]; exists {
			rh.idempotencyLock.RUnlock()
			log.Printf("📌 Idempotent request, returning existing route: %s", existingRouteID)
			// Return existing route
			rh.GetRoute(c, existingRouteID)
			return
		}
		rh.idempotencyLock.RUnlock()
	}

	// Get concurrency lock for operator+date
	lock := rh.getRouteLock(req.OperatorID, req.WorkDate)
	if !lock.TryLock() {
		log.Printf("⚠️  Route already being generated for %s on %s", req.OperatorID, req.WorkDate)
		c.JSON(http.StatusConflict, gin.H{"error": "Route is already being generated for this operator on this date"})
		return
	}
	defer lock.Unlock()

	// Check if route already exists
	var existingRoute models.Route
	if err := rh.db.Where("operator_id = ? AND work_date = ?", req.OperatorID, req.WorkDate).First(&existingRoute).Error; err == nil {
		log.Printf("⚠️  Route already exists: %s", existingRoute.ID)
		c.JSON(http.StatusConflict, gin.H{"error": "Route already exists for this operator on this date"})
		return
	}

	// Create new route
	routeID := uuid.New()
	route := models.Route{
		ID:         routeID,
		OperatorID: req.OperatorID,
		WorkDate:   req.WorkDate,
		ShiftID:    req.ShiftID,
		Status:     "generando",
		Distance:   0,
		Duration:   0,
		Polyline:   "",
	}

	if err := rh.db.Create(&route).Error; err != nil {
		log.Printf("❌ Failed to create route: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create route"})
		return
	}

	log.Printf("🆕 Created route: %s", routeID)

	// Store idempotency record
	if idempotencyKey != "" {
		rh.idempotencyLock.Lock()
		rh.idempotencyRecords[idempotencyKey] = routeID.String()
		rh.idempotencyLock.Unlock()
	}

	// Optimize route asynchronously
	go rh.optimizeRouteAsync(c.Request.Context(), routeID, req.OperatorID, req.WorkDate, req.TaskIDs)

	c.JSON(http.StatusCreated, models.RouteResponse{
		ID:         routeID.String(),
		OperatorID: req.OperatorID,
		WorkDate:   req.WorkDate,
		Status:     "generando",
		Distance:   0,
		Duration:   0,
	})
}

// optimizeRouteAsync runs optimization asynchronously
func (rh *RouteHandler) optimizeRouteAsync(ctx context.Context, routeID uuid.UUID, operatorID string, workDate string, taskIDs []string) {
	// Fetch tasks from database
	var tasks []models.Task
	if err := rh.db.Where("id IN ?", taskIDs).Find(&tasks).Error; err != nil {
		log.Printf("❌ Failed to fetch tasks: %v", err)
		rh.updateRouteStatus(routeID, "fallida")
		return
	}

	if len(tasks) == 0 {
		log.Printf("⚠️  No tasks found for route")
		rh.updateRouteStatus(routeID, "fallida")
		return
	}

	log.Printf("🔍 Optimizing route with %d tasks", len(tasks))

	// Sort tasks by priority
	tasks = optimizer.SortTasksByPriority(tasks)

	// Build distance matrix
	distanceMatrix := optimizer.BuildDistanceMatrix(tasks)

	// Check cache
	pointsHash := rh.distanceCache.GeneratePointsHash(tasks)
	cachedMatrix, found := rh.distanceCache.Get(pointsHash)
	if found {
		log.Printf("✅ Using cached distance matrix")
		distanceMatrix = cachedMatrix
	} else {
		// Store in cache
		rh.distanceCache.Set(pointsHash, distanceMatrix)
		log.Printf("💾 Stored distance matrix in cache")
	}

	// Optimize with timeout
	ctx, cancel := context.WithTimeout(ctx, time.Duration(rh.vrpOptimizer.solverTimeoutSeconds)*time.Second)
	defer cancel()

	optimizedRoute, err := rh.vrpOptimizer.OptimizeRoute(tasks, distanceMatrix, time.Now())
	if err != nil {
		log.Printf("❌ Optimization failed: %v", err)
		rh.updateRouteStatus(routeID, "fallida")
		return
	}

	// Create route steps
	for _, step := range optimizedRoute.Steps {
		routeStep := models.RouteStep{
			ID:            uuid.New(),
			RouteID:       routeID,
			TaskID:        step.TaskID,
			Seq:           step.Seq,
			ArrivalETA:    step.ArrivalETA,
			DepartureTime: step.DepartureTime,
			LegDistanceM:  step.LegDistanceM,
			LegDurationS:  step.LegDurationS,
		}
		if err := rh.db.Create(&routeStep).Error; err != nil {
			log.Printf("❌ Failed to create route step: %v", err)
			rh.updateRouteStatus(routeID, "fallida")
			return
		}
	}

	// Update route status
	status := "generada"
	if optimizedRoute.IsHeuristic {
		status = "heuristica"
	}

	if err := rh.db.Model(&models.Route{}).Where("id = ?", routeID).Updates(map[string]interface{}{
		"status":   status,
		"distance": optimizedRoute.TotalDistance,
		"duration": optimizedRoute.TotalDuration,
	}).Error; err != nil {
		log.Printf("❌ Failed to update route: %v", err)
		return
	}

	log.Printf("✅ Route optimized: %s (status: %s)", routeID, status)

	// Publish event
	event := messaging.RutaCalculadaEvent{
		RutaID:         routeID.String(),
		OperadorID:     operatorID,
		FechaLaboral:   workDate,
		DistanciaTotal: optimizedRoute.TotalDistance,
		TiempoTotal:    optimizedRoute.TotalDuration,
		EsHeuristica:   optimizedRoute.IsHeuristic,
		Timestamp:      time.Now().Format(time.RFC3339),
	}

	for _, step := range optimizedRoute.Steps {
		event.Pasos = append(event.Pasos, messaging.PasoCalculado{
			TareaID:        step.TaskID,
			Secuencia:      step.Seq,
			HoraLlegada:    step.ArrivalETA.Format(time.RFC3339),
			HoraSalida:     step.DepartureTime.Format(time.RFC3339),
			DistanciaTramo: step.LegDistanceM,
			TiempoTramo:    step.LegDurationS,
		})
	}

	err = rh.rabbitmq.PublishEvent(ctx, "tarea.ruta_calculada", event)
	if err != nil {
		log.Printf("❌ Failed to publish route event: %v", err)
	}
}

// GetRoute retrieves a route by ID
// GET /api/v1/routes/:id
// @Summary Get route by ID
// @Description Retrieve a route by its ID
// @Tags routes
// @Produce json
// @Param id path string true "Route ID"
// @Success 200 {object} models.RouteResponse
// @Failure 404 {object} map[string]string "Route not found"
// @Router /api/v1/routes/{id} [get]
// @Security BearerAuth
func (rh *RouteHandler) GetRoute(c *gin.Context, routeID string) {
	var route models.Route
	if err := rh.db.Where("id = ?", routeID).First(&route).Error; err != nil {
		log.Printf("❌ Route not found: %s", routeID)
		c.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
		return
	}

	// Fetch route steps
	var steps []models.RouteStep
	rh.db.Where("route_id = ?", routeID).Order("seq ASC").Find(&steps)

	c.JSON(http.StatusOK, models.RouteResponse{
		ID:         route.ID.String(),
		OperatorID: route.OperatorID,
		WorkDate:   route.WorkDate,
		Status:     route.Status,
		Distance:   route.Distance,
		Duration:   route.Duration,
		Steps:      steps,
	})
}

// GetRouteSteps retrieves steps of a route
// GET /api/v1/routes/:operator_id/:date/steps
// @Summary Get route steps
// @Description Retrieve all steps for an operator's route on a specific date
// @Tags routes
// @Produce json
// @Param operator_id path string true "Operator ID"
// @Param date path string true "Work date (YYYY-MM-DD)"
// @Success 200 {array} models.RouteStepResponse
// @Failure 404 {object} map[string]string "Route not found"
// @Router /api/v1/routes/{operator_id}/{date}/steps [get]
// @Security BearerAuth
func (rh *RouteHandler) GetRouteSteps(c *gin.Context) {
	operatorID := c.Param("operator_id")
	date := c.Param("date")

	var route models.Route
	if err := rh.db.Where("operator_id = ? AND work_date = ?", operatorID, date).First(&route).Error; err != nil {
		log.Printf("❌ Route not found for operator %s on %s", operatorID, date)
		c.JSON(http.StatusNotFound, gin.H{"error": "Route not found"})
		return
	}

	var steps []models.RouteStep
	if err := rh.db.Where("route_id = ?", route.ID).Order("seq ASC").Find(&steps).Error; err != nil {
		log.Printf("❌ Failed to fetch route steps: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch route steps"})
		return
	}

	responses := make([]models.RouteStepResponse, len(steps))
	for i, step := range steps {
		responses[i] = models.RouteStepResponse{
			TaskID:        step.TaskID,
			Seq:           step.Seq,
			ArrivalETA:    step.ArrivalETA,
			DepartureTime: step.DepartureTime,
			LegDistanceM:  step.LegDistanceM,
			LegDurationS:  step.LegDurationS,
		}
	}

	c.JSON(http.StatusOK, responses)
}

// updateRouteStatus updates the status of a route
func (rh *RouteHandler) updateRouteStatus(routeID uuid.UUID, status string) error {
	return rh.db.Model(&models.Route{}).Where("id = ?", routeID).Update("status", status).Error
}

// Health checks if the handler is healthy
func (rh *RouteHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "service": "route-service"})
}

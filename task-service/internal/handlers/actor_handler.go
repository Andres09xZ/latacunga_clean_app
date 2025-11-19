package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"task-service/internal/database"
	"task-service/internal/models"
)

// CreateActor allows the authenticated user to create an Actor record for themselves
// The user_id is taken from the JWT token and NOT from the request body.
//
// @Summary Create actor
// @Description Register an actor (worker/operator) associated to the authenticated user
// @Tags actors
// @Accept json
// @Produce json
// @Param request body object true "Actor payload (actor_type required)"
// @Success 201 {object} models.Actor
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Security BearerAuth
// @Router /actors [post]
func CreateActor(c *gin.Context) {
	var req struct {
		ActorType  string     `json:"actor_type" binding:"required"`
		Location   string     `json:"location"`
		ShiftStart *time.Time `json:"shift_start"`
		ShiftEnd   *time.Time `json:"shift_end"`
		Status     string     `json:"status"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract user id from token (set by JWTAuth middleware)
	userIDVal, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_id not found in token"})
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id in token"})
		return
	}
	uid, err := uuid.Parse(userIDStr)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user_id format"})
		return
	}

	actor := models.Actor{
		UserID:     &uid,
		ActorType:  req.ActorType,
		Location:   req.Location,
		ShiftStart: req.ShiftStart,
		ShiftEnd:   req.ShiftEnd,
		Status:     req.Status,
	}

	if actor.Status == "" {
		actor.Status = "ACTIVE"
	}

	if err := database.DB.Create(&actor).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create actor"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"actor": actor})
}

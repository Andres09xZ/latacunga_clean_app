package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/Andres09xZ/latacunga_clean_app/report-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/report-service/internal/models"
	"github.com/gin-gonic/gin"
	amqp "github.com/rabbitmq/amqp091-go"
)

type createReportRequest struct {
	Type        string `json:"type" binding:"required,oneof=acopio critico"`
	Description string `json:"description" binding:"required"`
	Location    string `json:"location,omitempty"` // WKT format: "POINT(lon lat)"
	PhotoURL    string `json:"photo_url,omitempty"`
}

type createBatchReportRequest []createReportRequest

func emitReportEvent(report models.Report) {
	rabbitURL := os.Getenv("RABBITMQ_URL")
	if rabbitURL == "" {
		log.Println("RABBITMQ_URL not set, skipping event emission")
		return
	}

	conn, err := amqp.Dial(rabbitURL)
	if err != nil {
		log.Printf("Failed to connect to RabbitMQ: %v", err)
		return
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Failed to open channel: %v", err)
		return
	}
	defer ch.Close()

	q, err := ch.QueueDeclare(
		"reports", // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Printf("Failed to declare queue: %v", err)
		return
	}

	body, err := json.Marshal(report)
	if err != nil {
		log.Printf("Failed to marshal report: %v", err)
		return
	}

	err = ch.Publish(
		"",     // exchange
		q.Name, // routing key
		false,  // mandatory
		false,  // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})
	if err != nil {
		log.Printf("Failed to publish message: %v", err)
	}
}

// CreateReport crea un nuevo reporte.
// @Summary Create a new report
// @Description Create a report from a user
// @Tags Reports
// @Accept json
// @Produce json
// @Param payload body createReportRequest true "Report payload"
// @Security BearerAuth
// @Success 201 {object} models.Report
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/reports [post]
func CreateReport(c *gin.Context) {
	var req createReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	report := models.Report{
		UserID:      userID.(string),
		Type:        req.Type,
		Description: req.Description,
		Location:    req.Location,
		PhotoURL:    req.PhotoURL,
		Status:      "Pendiente",
	}

	if err := database.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create report"})
		return
	}

	go emitReportEvent(report)

	c.JSON(http.StatusCreated, report)
}

// ListReports lista reportes (solo admin/operator).
// @Summary List reports
// @Description Get list of reports
// @Tags Reports
// @Produce json
// @Security BearerAuth
// @Success 200 {array} models.Report
// @Failure 500 {object} map[string]string
// @Router /api/v1/reports [get]
func ListReports(c *gin.Context) {
	var reports []models.Report
	if err := database.DB.Find(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db error"})
		return
	}
	c.JSON(http.StatusOK, reports)
}

// CreateBatchReports crea múltiples reportes en batch.
// @Summary Create batch reports
// @Description Create multiple reports from a user (for offline sync)
// @Tags Reports
// @Accept json
// @Produce json
// @Param payload body createBatchReportRequest true "Batch report payload"
// @Security BearerAuth
// @Success 201 {array} models.Report
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/reports/batch [post]
func CreateBatchReports(c *gin.Context) {
	var req createBatchReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user not authenticated"})
		return
	}

	var reports []models.Report
	for _, r := range req {
		report := models.Report{
			UserID:      userID.(string),
			Type:        r.Type,
			Description: r.Description,
			Location:    r.Location,
			PhotoURL:    r.PhotoURL,
			Status:      "Pendiente",
		}
		reports = append(reports, report)
	}

	if err := database.DB.Create(&reports).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create reports"})
		return
	}

	for _, r := range reports {
		go emitReportEvent(r)
	}

	c.JSON(http.StatusCreated, reports)
}

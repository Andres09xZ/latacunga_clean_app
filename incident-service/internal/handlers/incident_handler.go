package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/database"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/messaging"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/models"
	"github.com/Andres09xZ/latacunga_clean_app/incident-service/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CreateIncident crea un nuevo incidente
// @Summary Crear un nuevo incidente
// @Description Crea un incidente reportado por un ciudadano. Requiere autenticación JWT. El campo 'idempotency_key' es OPCIONAL y permite prevenir duplicados: si se envía la misma clave dos veces, se retorna el incidente existente en lugar de crear uno nuevo. Si no se proporciona, se permite la creación de múltiples incidentes (útil para desarrollo).
// @Tags Incidents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param incident body models.CreateIncidentRequest true "Datos del incidente a crear"
// @Success 201 {object} models.IncidentResponse "Incidente creado exitosamente"
// @Success 200 {object} models.IncidentResponse "Incidente ya existía (idempotencia activada con idempotency_key)"
// @Failure 400 {object} map[string]string "Solicitud inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 403 {object} map[string]string "Prohibido - solo ciudadanos pueden crear incidentes"
// @Failure 409 {object} map[string]string "Conflicto - idempotency_key usado pero recurso no encontrado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/incidents [post]
func CreateIncident(c *gin.Context) {
	var req models.CreateIncidentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	// ==== EXTRAER DATOS DEL JWT TOKEN ====
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing user_id in token"})
		return
	}

	role, exists := c.Get("role")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing role in token"})
		return
	}

	// Validar que solo ciudadanos pueden crear incidentes
	// Aceptar tanto "ciudadano" como "user" (del auth-service)
	reporterKind := role.(string)
	if reporterKind != "ciudadano" && reporterKind != "user" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Solo ciudadanos pueden crear incidentes",
			"role":  reporterKind,
		})
		return
	}

	// Normalizar el rol a "ciudadano" para la BD
	if reporterKind == "user" {
		reporterKind = "ciudadano"
	}

	reporterID := userID.(string)
	db := database.DB

	// ==== IDEMPOTENCIA: Verificar si ya existe (solo si se proporciona idempotency_key) ====
	// La idempotency_key es OPCIONAL. Si el cliente la proporciona, se verifica duplicados.
	// Si NO la proporciona, se crea el incidente normalmente (permitiendo duplicados si el cliente envía múltiples requests).
	idempotencyKey := req.IdempotencyKey
	if idempotencyKey != "" {
		// El cliente proporcionó una clave, verificar si ya existe
		var existingKey models.IdempotencyKey
		if err := db.Where("key = ?", idempotencyKey).First(&existingKey).Error; err == nil {
			// La clave ya existe, retornar el recurso existente
			if existingKey.ResourceID != nil {
				var existingIncident models.Incident
				if err := db.Where("id = ?", existingKey.ResourceID).First(&existingIncident).Error; err == nil {
					log.Printf("Idempotency: Returning existing incident %s for key %s", existingIncident.ID, idempotencyKey)
					response := convertIncidentToResponseWithKey(&existingIncident, idempotencyKey)
					c.JSON(http.StatusOK, response)
					return
				}
			}
			// Si la clave existe pero no el recurso, es un error
			c.JSON(http.StatusConflict, gin.H{"error": "Idempotency key already used but resource not found"})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			// Error de base de datos diferente a "no encontrado"
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
			return
		}
		// Si llegamos aquí, la clave NO existe y podemos crear el incidente
	}

	// ==== CREAR INCIDENTE ====
	// Capturar automáticamente la fecha del servidor en UTC
	incidentDay := time.Now().UTC().Truncate(24 * time.Hour)

	// Validar que latitude y longitude no sean nulos
	if req.Latitude == nil || req.Longitude == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude and Longitude are required"})
		return
	}

	// Validar que Type sea uno de los permitidos
	validTypes := []string{"punto_acopio", "zona_critica", "animal_muerto", "zona_reciclaje"}
	isValidType := false
	for _, t := range validTypes {
		if req.Type == t {
			isValidType = true
			break
		}
	}
	if !isValidType {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid type. Must be one of: punto_acopio, zona_critica, animal_muerto, zona_reciclaje",
		})
		return
	}

	// Construir punto geográfico: POINT(longitude latitude)
	location := fmt.Sprintf("POINT(%f %f)", *req.Longitude, *req.Latitude)

	incident := models.Incident{
		ID:           uuid.New().String(),
		ReporterKind: reporterKind,
		ReporterID:   &reporterID,
		Type:         models.IncidentType(req.Type),
		Description:  req.Description,
		Location:     location,
		Address:      req.Address,
		Status:       models.IncidentStatusNoValidado, // Estado inicial: "incidente_no_validado"
		IncidentDay:  incidentDay,
		PhotosCount:  0,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Registrar evento
	recordIncidentEvent(&incident, "incidente_creado")

	// Solicitar validacion
	if err := services.RequestValidation(&incident); err != nil {
		log.Printf("Warning: Failed to request validation: %v", err)
	}

	// Crear incidente en la base de datos
	if err := db.Create(&incident).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create incident: " + err.Error()})
		return
	}

	// Si se proporcionó una foto inicial, agregarla como adjunto
	if req.PhotoURL != nil && *req.PhotoURL != "" {
		attachment := models.IncidentAttachment{
			ID:         uuid.New().String(),
			IncidentID: incident.ID,
			FileURL:    *req.PhotoURL,
			CreatedAt:  time.Now(),
		}
		if err := db.Create(&attachment).Error; err == nil {
			incident.PhotosCount = 1
			db.Save(&incident)
		}
	}

	// ==== REGISTRAR IDEMPOTENCIA (solo si se proporcionó idempotency_key) ====
	if idempotencyKey != "" {
		idempotencyKeyRecord := models.IdempotencyKey{
			Key:        idempotencyKey,
			ResourceID: &incident.ID,
			CreatedAt:  time.Now(),
		}
		if err := db.Create(&idempotencyKeyRecord).Error; err != nil {
			log.Printf("Warning: Failed to save idempotency key: %v", err)
		} else {
			log.Printf("Idempotency: Registered key %s for incident %s", idempotencyKey, incident.ID)
		}
	}

	// ==== REGISTRAR EVENTO ====
	recordIncidentEvent(&incident, "incidente_pendiente")

	// ==== PUBLICAR EVENTO A RABBITMQ ====
	emitIncidentEvent(&incident, "incidente_pendiente")

	// Preparar respuesta (incluir idempotency_key solo si se proporcionó)
	var response models.IncidentResponse
	if idempotencyKey != "" {
		response = convertIncidentToResponseWithKey(&incident, idempotencyKey)
	} else {
		response = convertIncidentToResponse(&incident)
	}

	c.JSON(http.StatusCreated, response)
}

// GetIncident obtiene los detalles de un incidente por ID
// @Summary Obtener detalles de un incidente
// @Description Obtiene información detallada de un incidente incluyendo adjuntos y eventos
// @Tags Incidents
// @Produce json
// @Param id path string true "ID del incidente (UUID)"
// @Success 200 {object} models.IncidentResponse "Detalles del incidente"
// @Failure 400 {object} map[string]string "ID inválido"
// @Failure 404 {object} map[string]string "Incidente no encontrado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/incidents/{id} [get]
func GetIncident(c *gin.Context) {
	incidentID := c.Param("id")

	if _, err := uuid.Parse(incidentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid incident ID format"})
		return
	}

	db := database.DB

	var incident models.Incident
	if err := db.Preload("Attachments").Preload("Events").Where("id = ?", incidentID).First(&incident).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	response := convertIncidentToResponse(&incident)
	c.JSON(http.StatusOK, response)
}

// ListIncidents lista incidentes con paginación y filtros
// @Summary Listar incidentes
// @Description Obtiene una lista paginada de incidentes con filtros opcionales por tipo, estado y fecha
// @Tags Incidents
// @Produce json
// @Param page query int false "Número de página (default: 1)"
// @Param page_size query int false "Registros por página (default: 20, max: 100)"
// @Param type query string false "Filtrar por tipo: punto_acopio, zona_critica, animal_muerto, zona_reciclaje"
// @Param status query string false "Filtrar por estado: incidente_no_validado, emitido, valido, rechazado, convertido_en_tarea, cerrado"
// @Param from_date query string false "Fecha inicial (YYYY-MM-DD)"
// @Param to_date query string false "Fecha final (YYYY-MM-DD)"
// @Success 200 {object} map[string]interface{} "Lista de incidentes con metadatos de paginación"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/incidents [get]
func ListIncidents(c *gin.Context) {
	db := database.DB

	// Parámetros de paginación
	page := 1
	if p, err := strconv.Atoi(c.DefaultQuery("page", "1")); err == nil && p > 0 {
		page = p
	}

	pageSize := 20
	if ps, err := strconv.Atoi(c.DefaultQuery("page_size", "20")); err == nil && ps > 0 && ps <= 100 {
		pageSize = ps
	}

	offset := (page - 1) * pageSize

	// Filtros opcionales
	query := db.Model(&models.Incident{})
	if typeFilter := c.Query("type"); typeFilter != "" {
		query = query.Where("type = ?", typeFilter)
	}
	if statusFilter := c.Query("status"); statusFilter != "" {
		query = query.Where("status = ?", statusFilter)
	}

	var incidents []models.Incident
	var total int64

	// Contar total
	if err := query.Count(&total).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to count incidents: " + err.Error()})
		return
	}

	// Obtener página
	if err := query.
		Preload("Attachments").
		Offset(offset).
		Limit(pageSize).
		Order("created_at DESC").
		Find(&incidents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch incidents: " + err.Error()})
		return
	}

	// Convertir a respuestas
	responses := make([]models.IncidentResponse, 0, len(incidents))
	for _, inc := range incidents {
		responses = append(responses, convertIncidentToResponse(&inc))
	}

	totalPages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if totalPages == 0 {
		totalPages = 1
	}

	response := models.ListIncidentsResponse{
		Incidents:  responses,
		Total:      int(total),
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateIncidentStatus actualiza el estado de un incidente
// @Summary Actualizar estado de incidente
// @Description Actualiza el estado de un incidente. Solo operadores y administradores pueden realizar esta acción.
// @Tags Incidents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID del incidente (UUID)"
// @Param status body models.UpdateIncidentStatusRequest true "Nuevo estado del incidente"
// @Success 200 {object} models.IncidentResponse "Incidente actualizado"
// @Failure 400 {object} map[string]string "Solicitud inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 403 {object} map[string]string "Prohibido - se requiere rol de operador o admin"
// @Failure 404 {object} map[string]string "Incidente no encontrado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/incidents/{id}/status [put]
func UpdateIncidentStatus(c *gin.Context) {
	incidentID := c.Param("id")

	if _, err := uuid.Parse(incidentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid incident ID format"})
		return
	}

	var req models.UpdateIncidentStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	db := database.DB

	// Obtener incidente
	var incident models.Incident
	if err := db.Where("id = ?", incidentID).First(&incident).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	// Validar transición de estado
	//oldStatus := incident.Status
	newStatus := models.IncidentStatus(req.Status)

	/**if !isValidStatusTransition(incident.Status, newStatus) {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("Invalid status transition from %s to %s", oldStatus, newStatus),
		})
		return
	}**/

	// Actualizar estado
	incident.Status = newStatus
	incident.UpdatedAt = time.Now()

	if err := db.Save(&incident).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update incident: " + err.Error()})
		return
	}

	// Registrar evento
	recordIncidentEvent(&incident, fmt.Sprintf("incidente_status_changed_%s", req.Status))

	// Publicar evento
	emitIncidentEvent(&incident, "incidente_status_changed")

	response := convertIncidentToResponse(&incident)
	c.JSON(http.StatusOK, response)
}

// AddIncidentAttachment agrega una foto/archivo a un incidente
// @Summary Agregar adjunto a incidente
// @Description Agrega una foto o archivo como evidencia a un incidente existente. Requiere autenticación.
// @Tags Incidents
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "ID del incidente (UUID)"
// @Param attachment body models.AddIncidentAttachmentRequest true "Datos del adjunto"
// @Success 201 {object} map[string]interface{} "Adjunto agregado exitosamente"
// @Failure 400 {object} map[string]string "Solicitud inválida"
// @Failure 401 {object} map[string]string "No autorizado"
// @Failure 404 {object} map[string]string "Incidente no encontrado"
// @Failure 500 {object} map[string]string "Error interno del servidor"
// @Router /api/v1/incidents/{id}/attachments [post]
func AddIncidentAttachment(c *gin.Context) {
	incidentID := c.Param("id")

	if _, err := uuid.Parse(incidentID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid incident ID format"})
		return
	}

	var req models.AddIncidentAttachmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	db := database.DB

	// Verificar que el incidente exista
	var incident models.Incident
	if err := db.Where("id = ?", incidentID).First(&incident).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Incident not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error: " + err.Error()})
		return
	}

	// Crear attachment
	mimeType := req.MimeType
	sizeBytes := req.SizeBytes
	attachment := models.IncidentAttachment{
		ID:         uuid.New().String(),
		IncidentID: incidentID,
		FileURL:    req.FileURL,
		MimeType:   &mimeType,
		SizeBytes:  &sizeBytes,
		CreatedAt:  time.Now(),
	}

	if err := db.Create(&attachment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create attachment: " + err.Error()})
		return
	}

	// Actualizar contador de fotos
	incident.PhotosCount++
	incident.UpdatedAt = time.Now()
	if err := db.Save(&incident).Error; err != nil {
		log.Printf("Warning: Failed to update photos count: %v", err)
	}

	// Registrar evento
	recordIncidentEvent(&incident, "attachment_added")

	// Recargar incidente con attachments
	db.Preload("Attachments").Where("id = ?", incidentID).First(&incident)

	response := convertIncidentToResponse(&incident)
	c.JSON(http.StatusOK, response)
}

// ==== HELPER FUNCTIONS ====

// convertIncidentToResponse convierte un modelo Incident a IncidentResponse DTO
func convertIncidentToResponse(incident *models.Incident) models.IncidentResponse {
	return convertIncidentToResponseWithKey(incident, "")
}

// convertIncidentToResponseWithKey convierte un modelo Incident a IncidentResponse DTO con idempotency_key
func convertIncidentToResponseWithKey(incident *models.Incident, idempotencyKey string) models.IncidentResponse {
	// Extraer coordenadas de la geometría (formato POINT(lon lat))
	lat, lon := 0.0, 0.0
	fmt.Sscanf(incident.Location, "POINT(%f %f)", &lon, &lat)

	attachments := make([]models.IncidentAttachmentResponse, len(incident.Attachments))
	for i, att := range incident.Attachments {
		attachments[i] = models.IncidentAttachmentResponse{
			ID:        att.ID,
			FileURL:   att.FileURL,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
			CreatedAt: att.CreatedAt,
		}
	}

	return models.IncidentResponse{
		ID:             incident.ID,
		IdempotencyKey: idempotencyKey,
		ReporterKind:   incident.ReporterKind,
		ReporterID:     incident.ReporterID,
		Type:           string(incident.Type),
		Description:    incident.Description,
		Location: models.LocationResponse{
			Latitude:  lat,
			Longitude: lon,
		},
		Address:     incident.Address,
		Status:      string(incident.Status),
		IncidentDay: incident.IncidentDay,
		PhotosCount: incident.PhotosCount,
		Attachments: attachments,
		CreatedAt:   incident.CreatedAt,
		UpdatedAt:   incident.UpdatedAt,
	}
}

// isValidStatusTransition valida si la transición de estado es permitida
/**func isValidStatusTransition(from, to models.IncidentStatus) bool {
	validTransitions := map[models.IncidentStatus][]models.IncidentStatus{
		models.IncidentStatusEmitido: {
			models.IncidentStatusValido,
			models.IncidentStatusRechazado,
			models.IncidentStatusCerrado,
		},
		models.IncidentStatusValido: {
			models.IncidentStatusConvertidoEnTarea,
			models.IncidentStatusCerrado,
		},
		models.IncidentStatusRechazado: {
			models.IncidentStatusCerrado,
		},
		models.IncidentStatusConvertidoEnTarea: {
			models.IncidentStatusCerrado,
		},
		models.IncidentStatusCerrado: {},
	}

	if allowed, exists := validTransitions[from]; exists {
		for _, s := range allowed {
			if s == to {
				return true
			}
		}
	}
	return false
}**/

// recordIncidentEvent registra un evento en el histórico del incidente
func recordIncidentEvent(incident *models.Incident, eventType string) {
	event := models.IncidentEvent{
		ID:         uuid.New().String(),
		IncidentID: incident.ID,
		EventType:  eventType,
		Payload: json.RawMessage(fmt.Sprintf(
			`{"timestamp": "%s", "status": "%s"}`,
			time.Now().Format(time.RFC3339),
			incident.Status,
		)),
		CreatedAt: time.Now(),
	}

	if err := database.DB.Create(&event).Error; err != nil {
		log.Printf("Warning: Failed to record event: %v", err)
	}
}

// emitIncidentEvent publica un evento de incidente completo a RabbitMQ
func emitIncidentEvent(incident *models.Incident, eventType string) {
	// Extraer latitud y longitud del campo location (POINT(lng lat))
	var latitude, longitude float64
	fmt.Sscanf(incident.Location, "POINT(%f %f)", &longitude, &latitude)

	payload := map[string]interface{}{
		"id":            incident.ID,
		"reporter_kind": incident.ReporterKind,
		"reporter_id":   incident.ReporterID,
		"type":          incident.Type,
		"description":   incident.Description,
		"location": map[string]interface{}{
			"latitude":  latitude,
			"longitude": longitude,
		},
		"address":      incident.Address,
		"status":       incident.Status,
		"incident_day": incident.IncidentDay,
		"photos_count": incident.PhotosCount,
		"created_at":   incident.CreatedAt,
		"updated_at":   incident.UpdatedAt,
	}

	if err := messaging.PublishEvent(eventType, payload); err != nil {
		log.Printf("Warning: Failed to emit event to RabbitMQ: %v", err)
	} else {
		log.Printf("Incident event published successfully: type=%s, incident_id=%s", eventType, incident.ID)
	}
}

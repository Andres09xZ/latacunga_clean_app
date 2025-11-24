package models

// DTOs para las peticiones y respuestas de la API

// CompleteStopRequest representa el body para marcar una parada como completada
type CompleteStopRequest struct {
	Status  StopStatus `json:"status" binding:"required,oneof=RECOGIDO NO_RECOGIDO"`
	Comment string     `json:"comment,omitempty"`
}

// StartOrderRequest representa el body para iniciar una orden
type StartOrderRequest struct {
	StartLatitude  float64 `json:"start_latitude" binding:"required"`
	StartLongitude float64 `json:"start_longitude" binding:"required"`
}

// FinishOrderRequest representa el body para finalizar una orden
type FinishOrderRequest struct {
	EndLatitude  float64 `json:"end_latitude" binding:"required"`
	EndLongitude float64 `json:"end_longitude" binding:"required"`
}

// WorkOrderResponse representa la respuesta completa de una orden con sus paradas
type WorkOrderResponse struct {
	WorkOrder
	Stops []WorkOrderStop `json:"stops"`
}

// ErrorResponse representa una respuesta de error
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse representa una respuesta exitosa
type SuccessResponse struct {
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

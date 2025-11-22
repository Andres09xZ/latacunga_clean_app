package models

import (
	"time"

	"github.com/google/uuid"
)

// IncidentePendiente representa un incidente almacenado localmente para validación
type IncidentePendiente struct {
	IncidenteID  uuid.UUID `gorm:"type:uuid;primaryKey;column:incidente_id" json:"incidente_id"`
	Tipo         string    `gorm:"column:tipo;size:50;not null" json:"tipo"`
	Descripcion  string    `gorm:"column:descripcion;type:text;not null" json:"descripcion"`
	CiudadanoID  uuid.UUID `gorm:"type:uuid;column:ciudadano_id;not null" json:"ciudadano_id"`
	Estado       string    `gorm:"column:estado;size:50;default:'pendiente_validacion'" json:"estado"`
	FechaEvento  time.Time `gorm:"column:fecha_evento;not null" json:"fecha_evento"`
	DiaIncidente time.Time `gorm:"column:dia_incidente;type:date;not null" json:"dia_incidente"`
	NumFotos     int       `gorm:"column:num_fotos;default:0" json:"num_fotos"`
	Direccion    string    `gorm:"column:direccion;type:text" json:"direccion"`
	Latitud      float64   `gorm:"column:latitud;type:decimal(10,8);not null" json:"latitud"`
	Longitud     float64   `gorm:"column:longitud;type:decimal(11,8);not null" json:"longitud"`

	// Datos de validación
	ValidadoPor     *string    `gorm:"column:validado_por;size:100" json:"validado_por,omitempty"`
	FechaValidacion *time.Time `gorm:"column:fecha_validacion" json:"fecha_validacion,omitempty"`
	NotasValidacion *string    `gorm:"column:notas_validacion;type:text" json:"notas_validacion,omitempty"`

	// Auditoría
	RecibidoEn    time.Time `gorm:"column:recibido_en;autoCreateTime" json:"recibido_en"`
	ActualizadoEn time.Time `gorm:"column:actualizado_en;autoUpdateTime" json:"actualizado_en"`
}

// TableName especifica el nombre de la tabla
func (IncidentePendiente) TableName() string {
	return "validacion.incidentes_pendientes"
}

// IncidentePendienteResponse es la respuesta API
type IncidentePendienteResponse struct {
	IncidenteID     string  `json:"incidente_id"`
	Tipo            string  `json:"tipo"`
	Descripcion     string  `json:"descripcion"`
	CiudadanoID     string  `json:"ciudadano_id"`
	Estado          string  `json:"estado"`
	FechaEvento     string  `json:"fecha_evento"`
	DiaIncidente    string  `json:"dia_incidente"`
	NumFotos        int     `json:"num_fotos"`
	Direccion       string  `json:"direccion"`
	Latitud         float64 `json:"latitud"`
	Longitud        float64 `json:"longitud"`
	ValidadoPor     *string `json:"validado_por,omitempty"`
	FechaValidacion *string `json:"fecha_validacion,omitempty"`
	NotasValidacion *string `json:"notas_validacion,omitempty"`
	RecibidoEn      string  `json:"recibido_en"`
	ActualizadoEn   string  `json:"actualizado_en"`
}

// ToResponse convierte el modelo a respuesta API
func (i *IncidentePendiente) ToResponse() IncidentePendienteResponse {
	resp := IncidentePendienteResponse{
		IncidenteID:     i.IncidenteID.String(),
		Tipo:            i.Tipo,
		Descripcion:     i.Descripcion,
		CiudadanoID:     i.CiudadanoID.String(),
		Estado:          i.Estado,
		FechaEvento:     i.FechaEvento.Format(time.RFC3339),
		DiaIncidente:    i.DiaIncidente.Format("2006-01-02"),
		NumFotos:        i.NumFotos,
		Direccion:       i.Direccion,
		Latitud:         i.Latitud,
		Longitud:        i.Longitud,
		ValidadoPor:     i.ValidadoPor,
		NotasValidacion: i.NotasValidacion,
		RecibidoEn:      i.RecibidoEn.Format(time.RFC3339),
		ActualizadoEn:   i.ActualizadoEn.Format(time.RFC3339),
	}

	if i.FechaValidacion != nil {
		fechaStr := i.FechaValidacion.Format(time.RFC3339)
		resp.FechaValidacion = &fechaStr
	}

	return resp
}

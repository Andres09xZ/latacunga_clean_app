package service

import "log"

// OrchestratorAdapter adapta el Orchestrator para cumplir con la interfaz de TriggerLogic
type OrchestratorAdapter struct {
	orchestrator *Orchestrator
}

// NewOrchestratorAdapter crea un nuevo adaptador
func NewOrchestratorAdapter(orchestrator *Orchestrator) *OrchestratorAdapter {
	return &OrchestratorAdapter{
		orchestrator: orchestrator,
	}
}

// EvaluateAndTrigger cumple con la interfaz esperada por PlanningService
func (oa *OrchestratorAdapter) EvaluateAndTrigger(zoneID int, newScore int, threshold int, zoneName string) error {
	// Verificar si se alcanzó el umbral
	if newScore < threshold {
		log.Printf("⏳ Zona %d (%s): Puntaje %d no alcanza umbral %d", zoneID, zoneName, newScore, threshold)
		return nil
	}

	log.Printf("🚨 Zona %d (%s): Umbral alcanzado! Score=%d >= Threshold=%d", zoneID, zoneName, newScore, threshold)

	// Llamar al orchestrator para ejecutar la saga
	return oa.orchestrator.TriggerZone(zoneID, zoneName)
}

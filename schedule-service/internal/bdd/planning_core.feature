# language: es
Característica: Núcleo de Planificación (Planning Core)
  Como sistema gestor de residuos de Latacunga
  Quiero ubicar incidentes en las 5 zonas macro y acumular gravedad
  Para activar las rutas en los horarios correctos (Nocturno/Diurno/Rural)

  Antecedentes: Configuración Geográfica
    Dado que el sistema tiene cargado el GeoJSON con las siguientes zonas:
      | nombre          | horario               |
      | URBANO_CENTRAL  | NOCTURNO (21:00)      |
      | URBANO_NORTE    | DIURNO (Mar-Jue-Sab)  |
      | URBANO_SUR      | DIURNO (Lun-Mie-Vie)  |
      | RURAL_NORTE     | RURAL (Rutas 2,3,4)   |
      | RURAL_SUR       | RURAL (Rutas 1,5)     |
    Y el umbral de activación por defecto es 50 puntos

  Escenario: Incidente en el Centro Histórico
    Cuando llega un incidente validado con coordenadas lat: -0.930, lon: -78.615
    Entonces el sistema debe detectar que cae en el polígono "URBANO_CENTRAL"
    Y debe identificar que el horario base es "21:00"

  Escenario: Acumulación de puntaje sin disparo
    Dado que la zona "URBANO_CENTRAL" tiene un puntaje actual de 40
    Cuando llega un incidente de tipo "SENSOR_LLENO" (valor 5 pts) en esa zona
    Entonces el puntaje de la zona sube a 45
    Y el estado de la zona permanece en "ACUMULANDO"
    Y NO se emite ninguna solicitud de recursos

  Escenario: Superación de Umbral y Activación
    Dado que la zona "URBANO_CENTRAL" tiene un puntaje actual de 48
    Cuando llega un incidente de tipo "SENSOR_LLENO" (valor 5 pts) en esa zona
    Entonces el puntaje de la zona sube a 53
    Y el sistema detecta que se superó el umbral (53 > 50)
    Y el sistema debe emitir el evento "planning.resource.requested.v1"
    Y la solicitud debe pedir un camión para "HOY a las 21:00"

  Escenario: Incidente en Zona Rural Norte
    Cuando llega un incidente validado con coordenadas lat: -0.850, lon: -78.610
    Entonces el sistema debe detectar que cae en "RURAL_NORTE"
    Y el sistema debe asignar la fecha según el calendario de "Rutas 2,3,4"

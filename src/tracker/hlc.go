package tracker

// tracker/hlc.go
// Implementación de Hybrid Logical Clock (HLC) para sincronización de tiempo
// en sistemas distribuidos sin depender de relojes físicos sincronizados.

import (
	"encoding/json"
	"fmt"
	"time"
)

// HLC (Hybrid Logical Clock) combina tiempo físico con un contador lógico
// para establecer orden causal entre eventos en un sistema distribuido.
//
// Componentes:
// - PhysicalTime: Tiempo físico en milisegundos desde epoch (del reloj del sistema)
// - LogicalTime: Contador lógico que se incrementa para eventos concurrentes
// - NodeID: Identificador del nodo que generó este timestamp (para desempate)
type HLC struct {
	PhysicalTime int64  `json:"pt"` // milisegundos desde epoch
	LogicalTime  int64  `json:"lt"` // contador lógico
	NodeID       string `json:"node_id"`
}

// NewHLC crea un nuevo HLC inicializado con el tiempo actual del sistema.
func NewHLC(nodeID string) *HLC {
	return &HLC{
		PhysicalTime: time.Now().UnixMilli(),
		LogicalTime:  0,
		NodeID:       nodeID,
	}
}

// Update actualiza el HLC según las reglas del algoritmo:
//
// Caso 1: Evento local (msgHLC == nil)
//   - Avanza PhysicalTime al tiempo actual del sistema
//   - Incrementa LogicalTime en 1
//
// Caso 2: Recibe mensaje con PhysicalTime mayor (del "futuro")
//   - Adopta el PhysicalTime del mensaje
//   - Toma el máximo LogicalTime y lo incrementa en 1
//
// Caso 3: Recibe mensaje con PhysicalTime menor o igual (del "pasado")
//   - Mantiene el PhysicalTime local
//   - Toma el máximo LogicalTime y lo incrementa en 1
func (h *HLC) Update(msgHLC *HLC) {
	nowPhysical := time.Now().UnixMilli()

	if msgHLC == nil {
		// Evento local: avanzar al tiempo físico actual
		if nowPhysical > h.PhysicalTime {
			h.PhysicalTime = nowPhysical
			h.LogicalTime = 0
		} else {
			// El reloj del sistema retrocedió o está igual
			h.LogicalTime++
		}
	} else {
		// Evento de mensaje recibido: aplicar máximo de tres valores
		maxPT := max3(h.PhysicalTime, msgHLC.PhysicalTime, nowPhysical)

		if maxPT == h.PhysicalTime && maxPT == msgHLC.PhysicalTime {
			// Ambos tienen el mismo tiempo físico
			h.LogicalTime = max(h.LogicalTime, msgHLC.LogicalTime) + 1
		} else if maxPT == msgHLC.PhysicalTime {
			// El mensaje viene del "futuro"
			h.PhysicalTime = msgHLC.PhysicalTime
			h.LogicalTime = msgHLC.LogicalTime + 1
		} else if maxPT == nowPhysical {
			// El tiempo actual del sistema es el mayor
			h.PhysicalTime = nowPhysical
			h.LogicalTime = max(h.LogicalTime, msgHLC.LogicalTime) + 1
		} else {
			// El tiempo local es el mayor
			h.LogicalTime = max(h.LogicalTime, msgHLC.LogicalTime) + 1
		}
	}
}

// Clone crea una copia independiente del HLC.
func (h *HLC) Clone() HLC {
	return HLC{
		PhysicalTime: h.PhysicalTime,
		LogicalTime:  h.LogicalTime,
		NodeID:       h.NodeID,
	}
}

// After retorna true si h es posterior a other en orden causal.
// Criterios de comparación (en orden de prioridad):
// 1. PhysicalTime mayor → es posterior
// 2. Si PhysicalTime igual, LogicalTime mayor → es posterior
// 3. Si ambos iguales, NodeID alfabéticamente mayor → es posterior
func (h *HLC) After(other HLC) bool {
	if h.PhysicalTime != other.PhysicalTime {
		return h.PhysicalTime > other.PhysicalTime
	}
	if h.LogicalTime != other.LogicalTime {
		return h.LogicalTime > other.LogicalTime
	}
	return h.NodeID > other.NodeID
}

// Before retorna true si h es anterior a other en orden causal.
func (h *HLC) Before(other HLC) bool {
	if h.PhysicalTime != other.PhysicalTime {
		return h.PhysicalTime < other.PhysicalTime
	}
	if h.LogicalTime != other.LogicalTime {
		return h.LogicalTime < other.LogicalTime
	}
	return h.NodeID < other.NodeID
}

// Equal retorna true si ambos HLCs son idénticos.
func (h *HLC) Equal(other HLC) bool {
	return h.PhysicalTime == other.PhysicalTime &&
		h.LogicalTime == other.LogicalTime &&
		h.NodeID == other.NodeID
}

// String retorna una representación legible del HLC.
func (h *HLC) String() string {
	return fmt.Sprintf("HLC{pt:%d, lt:%d, node:%s}", h.PhysicalTime, h.LogicalTime, h.NodeID)
}

// MarshalJSON serializa el HLC a JSON.
func (h *HLC) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		PT   int64  `json:"pt"`
		LT   int64  `json:"lt"`
		Node string `json:"node_id"`
	}{
		PT:   h.PhysicalTime,
		LT:   h.LogicalTime,
		Node: h.NodeID,
	})
}

// UnmarshalJSON deserializa el HLC desde JSON.
func (h *HLC) UnmarshalJSON(data []byte) error {
	aux := struct {
		PT   int64  `json:"pt"`
		LT   int64  `json:"lt"`
		Node string `json:"node_id"`
	}{}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	h.PhysicalTime = aux.PT
	h.LogicalTime = aux.LT
	h.NodeID = aux.Node
	return nil
}

// SubtractDuration resta una duración al PhysicalTime del HLC.
// Útil para calcular umbrales de tiempo (ej: hace 10 minutos).
func (h *HLC) SubtractDuration(d time.Duration) HLC {
	return HLC{
		PhysicalTime: h.PhysicalTime - d.Milliseconds(),
		LogicalTime:  h.LogicalTime,
		NodeID:       h.NodeID,
	}
}

// Funciones auxiliares

func max(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func max3(a, b, c int64) int64 {
	return max(max(a, b), c)
}

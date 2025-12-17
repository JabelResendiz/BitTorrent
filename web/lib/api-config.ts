/**
 * Configuración del API Backend
 * 
 * IMPORTANTE: Si cambias el puerto del backend (api/config.go),
 * también debes actualizarlo aquí.
 */

export const API_CONFIG = {
  // URL base del API (cambiar si el backend usa otro puerto)
  BASE_URL: 'http://localhost:7000',
  
  // WebSocket URL
  WS_URL: 'ws://localhost:7000',
  
  // Endpoints
  ENDPOINTS: {
    CONTAINERS: '/api/containers',
    TORRENTS: '/api/torrents',
    NETWORKS: '/api/networks',
    HEALTH: '/health',
  }
} as const;

// Helper para construir URLs completas
export const getApiUrl = (endpoint: string) => {
  return `${API_CONFIG.BASE_URL}${endpoint}`;
};

export const getWsUrl = (path: string) => {
  return `${API_CONFIG.WS_URL}${path}`;
};

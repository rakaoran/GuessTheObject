// Environment-aware configuration
const isDev = import.meta.env.MODE === 'development';

// Base URLs
export const API_BASE_URL = isDev
  ? 'http://localhost:5000'
  : 'https://api.gto.rakaoran.dev';

export const WS_BASE_URL = isDev
  ? 'ws://localhost:5000'
  : 'wss://api.gto.rakaoran.dev';

// API Endpoints
export const endpoints = {
  login: `${API_BASE_URL}/auth/login`,
  signup: `${API_BASE_URL}/auth/signup`,
  logout: `${API_BASE_URL}/auth/logout`,
  refresh: `${API_BASE_URL}/auth/refresh`,
  publicGames: `${API_BASE_URL}/game/games`,
};

// WebSocket Endpoints
export const wsEndpoints = {
  createGame: `${WS_BASE_URL}/game/create`,
  joinRoom: (roomId: string) => `${WS_BASE_URL}/game/join/${roomId}`,
};
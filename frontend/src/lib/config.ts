import { dev } from '$app/environment';

// API base URL - use Go backend in dev, same origin in production
export const API_BASE_URL = dev ? 'http://127.0.0.1:8000' : '';

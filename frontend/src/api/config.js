// Dynamic API Configuration
// In production (ZeroTier/EC2), REACT_APP_BACKEND_URL will be set.
// In development, it falls back to localhost.

export const API_BASE_URL = import.meta.env.VITE_BACKEND_URL || "http://localhost:8080";

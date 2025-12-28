import axios from 'axios';

// Create axios instance with base configuration
export const api = axios.create({
    baseURL: process.env.REACT_APP_API_URL || 'http://localhost:8080',
    timeout: 30000, // 30 seconds timeout for cold storage operations
    headers: {
        'Content-Type': 'application/json',
    },
});

// Request interceptor to add authentication token
api.interceptors.request.use(
    (config) => {
        const token = localStorage.getItem('auth_token');
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);

// Response interceptor for error handling
api.interceptors.response.use(
    (response) => response,
    (error) => {
        if (error.response?.status === 401) {
            // Clear token and redirect to login
            localStorage.removeItem('auth_token');
            window.location.href = '/login';
        }
        return Promise.reject(error);
    }
);

export default api;
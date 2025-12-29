// Base API configuration
const baseURL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const timeout = 30000; // 30 seconds timeout for cold storage operations

// Fetch wrapper with authentication and error handling
class APIClient {
    private baseURL: string;
    private timeout: number;

    constructor(baseURL: string, timeout: number) {
        this.baseURL = baseURL;
        this.timeout = timeout;
    }

    private async request(endpoint: string, options: RequestInit = {}): Promise<any> {
        const url = `${this.baseURL}${endpoint}`;

        // Get auth token if available
        const token = typeof window !== 'undefined' ? localStorage.getItem('auth_token') : null;

        // Setup headers
        const headers = {
            'Content-Type': 'application/json',
            ...(options.headers || {}),
        };

        if (token) {
            headers.Authorization = `Bearer ${token}`;
        }

        // Create AbortController for timeout
        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.timeout);

        try {
            const response = await fetch(url, {
                ...options,
                headers,
                signal: controller.signal,
            });

            clearTimeout(timeoutId);

            // Handle 401 unauthorized
            if (response.status === 401) {
                if (typeof window !== 'undefined') {
                    localStorage.removeItem('auth_token');
                    window.location.href = '/';
                }
                throw new Error('Unauthorized');
            }

            // Parse JSON response
            const data = await response.json();

            if (!response.ok) {
                const error = new Error(data.message || 'Request failed');
                (error as any).response = { status: response.status, data };
                throw error;
            }

            return { data, status: response.status };
        } catch (error) {
            clearTimeout(timeoutId);
            if ((error as Error).name === 'AbortError') {
                throw new Error('Request timeout');
            }
            throw error;
        }
    }

    async get(endpoint: string, options?: RequestInit) {
        return this.request(endpoint, { ...options, method: 'GET' });
    }

    async post(endpoint: string, data?: any, options?: RequestInit) {
        return this.request(endpoint, {
            ...options,
            method: 'POST',
            body: data ? JSON.stringify(data) : undefined,
        });
    }

    async put(endpoint: string, data?: any, options?: RequestInit) {
        return this.request(endpoint, {
            ...options,
            method: 'PUT',
            body: data ? JSON.stringify(data) : undefined,
        });
    }

    async delete(endpoint: string, options?: RequestInit) {
        return this.request(endpoint, { ...options, method: 'DELETE' });
    }
}

// Export API instance
export const api = new APIClient(baseURL, timeout);
export default api;
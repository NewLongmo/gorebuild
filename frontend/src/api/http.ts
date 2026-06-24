import axios from 'axios';

export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface PageResult<T> {
  items: T[];
  total: number;
  page: number;
  perPage: number;
}

export const http = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: Number(import.meta.env.VITE_API_TIMEOUT_MS || 60000),
});

let unauthorizedHandler: (() => void) | undefined;

export function setUnauthorizedHandler(handler: () => void) {
  unauthorizedHandler = handler;
}

function handleUnauthorized() {
  setAuthToken('');
  unauthorizedHandler?.();
}

export function getAuthToken() {
  return localStorage.getItem('dw0rdwk_token') || '';
}

export function setAuthToken(token: string) {
  if (token) {
    localStorage.setItem('dw0rdwk_token', token);
  } else {
    localStorage.removeItem('dw0rdwk_token');
  }
}

http.interceptors.request.use((config) => {
  const token = getAuthToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

http.interceptors.response.use(
  (response) => {
    const payload = response.data as ApiResponse<unknown>;
    if (payload && payload.code !== 0) {
      if (payload.code === 401) {
        handleUnauthorized();
      }
      return Promise.reject(new Error(payload.message || '请求失败'));
    }
    return response;
  },
  (error) => {
    if (error?.response?.status === 401) {
      handleUnauthorized();
    }
    const payload = error?.response?.data as Partial<ApiResponse<unknown>> | undefined;
    if (payload?.message) {
      return Promise.reject(new Error(payload.message));
    }
    return Promise.reject(error);
  },
);

export async function getData<T>(url: string, params?: Record<string, unknown>): Promise<T> {
  const response = await http.get<ApiResponse<T>>(url, { params });
  return response.data.data;
}

export async function postData<T>(url: string, data?: Record<string, unknown>): Promise<T> {
  const response = await http.post<ApiResponse<T>>(url, data);
  return response.data.data;
}

export async function patchData<T>(url: string, data?: Record<string, unknown>): Promise<T> {
  const response = await http.patch<ApiResponse<T>>(url, data);
  return response.data.data;
}

export async function putData<T>(url: string, data?: Record<string, unknown>): Promise<T> {
  const response = await http.put<ApiResponse<T>>(url, data);
  return response.data.data;
}

export async function deleteData<T>(url: string): Promise<T> {
  const response = await http.delete<ApiResponse<T>>(url);
  return response.data.data;
}

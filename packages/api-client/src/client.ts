import type { ApiErrorResponse } from './types';

let apiBase = '/api';

export function setApiBase(url: string): void {
  apiBase = url;
}

export function getApiBase(): string {
  return apiBase;
}

let authToken: string | null = null;

export function setAuthToken(token: string | null): void {
  authToken = token;
}

export function getAuthToken(): string | null {
  return authToken;
}

export class ApiError extends Error {
  code: string;
  status: number;

  constructor(code: string, message: string, status: number) {
    super(message);
    this.name = 'ApiError';
    this.code = code;
    this.status = status;
  }
}

async function handleResponse<T>(response: Response): Promise<T> {
  if (!response.ok) {
    const data = (await response.json()) as ApiErrorResponse;
    throw new ApiError(
      data.error?.code || 'UNKNOWN_ERROR',
      data.error?.message || 'An unknown error occurred',
      response.status,
    );
  }
  return response.json() as Promise<T>;
}

export function authHeaders(): Record<string, string> {
  const headers: Record<string, string> = {};
  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }
  return headers;
}

export async function get<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${apiBase}${endpoint}`, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
    },
  });
  return handleResponse<T>(response);
}

export async function post<T>(endpoint: string, data?: unknown): Promise<T> {
  const response = await fetch(`${apiBase}${endpoint}`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
    },
    body: data ? JSON.stringify(data) : undefined,
  });
  return handleResponse<T>(response);
}

export async function uploadFile(
  endpoint: string,
  file: File,
  fields?: Record<string, string>,
): Promise<unknown> {
  const formData = new FormData();
  if (fields) {
    for (const [key, value] of Object.entries(fields)) {
      formData.append(key, value);
    }
  }
  formData.append('file', file);

  const response = await fetch(`${apiBase}${endpoint}`, {
    method: 'POST',
    headers: authHeaders(),
    body: formData,
  });
  return handleResponse(response);
}

export async function del<T>(endpoint: string): Promise<T> {
  const response = await fetch(`${apiBase}${endpoint}`, {
    method: 'DELETE',
    headers: {
      'Content-Type': 'application/json',
      ...authHeaders(),
    },
  });
  return handleResponse<T>(response);
}

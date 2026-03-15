import createClient from 'openapi-fetch';
import type { paths } from '../generated/schema';
import type { ApiErrorResponse } from './types';

let apiBase = '/api';
let apiClient = createApiClient(apiBase);

function createApiClient(baseUrl: string) {
  const client = createClient<paths>({ baseUrl });
  client.use({
    async onRequest({ request }) {
      const token = getAuthToken();
      if (token) {
        request.headers.set('Authorization', `Bearer ${token}`);
      }
      return request;
    },
  });
  return client;
}

export { apiClient };

export function setApiBase(url: string): void {
  apiBase = url;
  apiClient = createApiClient(url);
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

export async function throwIfError<T>(
  promise: Promise<{ data?: T; error?: unknown; response: Response }>,
): Promise<T> {
  const { data, error, response } = await promise;
  if (error) {
    let code = 'UNKNOWN_ERROR';
    let message = 'An unknown error occurred';
    if (typeof error === 'object' && error !== null) {
      const apiErr = error as ApiErrorResponse;
      code = apiErr.error?.code || code;
      message = apiErr.error?.message || message;
    }
    throw new ApiError(code, message, response.status);
  }
  if (data === undefined) {
    throw new ApiError('EMPTY_RESPONSE', 'No data returned', response.status);
  }
  return data;
}

// openapi-fetch lacks typed multipart support, so we bypass body type
// checking with `as never` and provide a custom serializer that returns
// the FormData directly instead of JSON-encoding it.
export function multipartRequest(formData: FormData) {
  return {
    body: formData as never,
    bodySerializer: () => formData,
  };
}

export function authHeaders(): Record<string, string> {
  const headers: Record<string, string> = {};
  if (authToken) {
    headers['Authorization'] = `Bearer ${authToken}`;
  }
  return headers;
}

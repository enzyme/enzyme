import { get, post, type User, type WorkspaceSummary } from '@feather/api-client';

export interface LoginInput {
  email: string;
  password: string;
}

export interface RegisterInput {
  email: string;
  password: string;
  display_name: string;
}

export interface AuthResponse {
  user: User;
}

export interface MeResponse {
  user: User;
  workspaces?: WorkspaceSummary[];
}

export const authApi = {
  login: (input: LoginInput) =>
    post<AuthResponse>('/auth/login', input),

  register: (input: RegisterInput) =>
    post<AuthResponse>('/auth/register', input),

  logout: () =>
    post<{ success: boolean }>('/auth/logout'),

  me: () =>
    get<MeResponse>('/auth/me'),

  forgotPassword: (email: string) =>
    post<{ success: boolean; message: string }>('/auth/forgot-password', { email }),

  resetPassword: (token: string, new_password: string) =>
    post<{ success: boolean }>('/auth/reset-password', { token, new_password }),
};

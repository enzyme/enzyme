import { apiClient, throwIfError } from '../client';
import type { LoginInput, RegisterDeviceTokenInput, RegisterInput } from '../types';

export const authApi = {
  login: (input: LoginInput) => throwIfError(apiClient.POST('/auth/login', { body: input })),

  register: (input: RegisterInput) =>
    throwIfError(apiClient.POST('/auth/register', { body: input })),

  logout: () => throwIfError(apiClient.POST('/auth/logout')),

  me: () => throwIfError(apiClient.GET('/auth/me')),

  forgotPassword: (email: string) =>
    throwIfError(apiClient.POST('/auth/forgot-password', { body: { email } })),

  resetPassword: (token: string, new_password: string) =>
    throwIfError(apiClient.POST('/auth/reset-password', { body: { token, new_password } })),

  verifyEmail: (token: string) =>
    throwIfError(apiClient.POST('/auth/verify-email', { body: { token } })),

  resendVerification: () => throwIfError(apiClient.POST('/auth/resend-verification')),

  registerDeviceToken: (input: RegisterDeviceTokenInput) =>
    throwIfError(apiClient.POST('/auth/device-tokens', { body: input })),

  unregisterDeviceToken: (id: string) =>
    throwIfError(apiClient.DELETE('/auth/device-tokens/{id}', { params: { path: { id } } })),
};

import { api } from '@/shared/api/client';
import { ApiError } from '@/shared/api/errors';
import type { LoginValues } from '../model/schemas';
import type { AuthSession, TenantOption } from './types';

/** A login opens a session, or asks which company to enter when the email is in several. */
export type LoginResult = { session: AuthSession } | { tenants: TenantOption[] };

/** POST /auth/login. Any other failure is thrown as an ApiError. */
export async function login(credentials: LoginValues, tenantId?: string): Promise<LoginResult> {
  const { data, error, response } = await api.POST('/api/v1/auth/login', {
    body: tenantId ? { ...credentials, tenant_id: tenantId } : credentials,
  });
  if (data) return { session: data };
  if ('tenants' in error) return { tenants: error.tenants };
  throw new ApiError(response.status, error);
}

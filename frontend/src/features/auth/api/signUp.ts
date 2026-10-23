import { api } from '@/shared/api/client';
import { unwrap } from '@/shared/api/errors';
import type { SignUpValues } from '../model/schemas';

/** POST /auth/signup: the new company, its owner and their session. Failures are thrown as an ApiError. */
export async function signUp(values: SignUpValues) {
  return unwrap(await api.POST('/api/v1/auth/signup', { body: values }));
}

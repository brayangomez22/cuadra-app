import { useState } from 'react';
import type { UseFormSetError } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { login } from '../../../api/login';
import type { TenantOption } from '../../../api/types';
import type { LoginValues } from '../../../model/schemas';
import { useAuth } from '../../../session/useAuth';

// The same email and password are valid in several companies: the user picks one.
type TenantChoice = { credentials: LoginValues; tenants: TenantOption[] };

// No error of POST /auth/login belongs to a field: a wrong password must not say which one failed.
const noFields = {};

/**
 * The login and, when the email is in several companies, the choice of one. A
 * successful login stores the session; GuestOnly then leads to the app.
 */
export function useLogin(setError: UseFormSetError<LoginValues>) {
  const { startSession } = useAuth();
  const { formError, handleError, clearFormError } = useApiFormErrors(setError, noFields);
  const [choice, setChoice] = useState<TenantChoice | null>(null);
  const [pendingTenant, setPendingTenant] = useState<string | null>(null);

  async function submit(credentials: LoginValues, tenantId?: string) {
    clearFormError();
    try {
      const result = await login(credentials, tenantId);
      if ('session' in result) startSession(result.session);
      else setChoice({ credentials, tenants: result.tenants });
    } catch (error) {
      handleError(error);
    }
  }

  async function chooseTenant(tenantId: string) {
    if (!choice) return;
    setPendingTenant(tenantId);
    await submit(choice.credentials, tenantId);
    setPendingTenant(null);
  }

  function back() {
    setChoice(null);
    clearFormError();
  }

  return { formError, choice, pendingTenant, submit, chooseTenant, back };
}

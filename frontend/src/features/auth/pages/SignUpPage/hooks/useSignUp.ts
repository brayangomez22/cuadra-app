import type { UseFormSetError } from 'react-hook-form';
import { useApiFormErrors, type FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { signUp } from '../../../api/signUp';
import type { SignUpValues } from '../../../model/schemas';
import { useAuth } from '../../../session/useAuth';

// The field each 422 code of POST /auth/signup belongs to (api/openapi.yaml).
const fieldByErrorCode: FieldsByErrorCode<SignUpValues> = {
  invalid_tenant_name: 'tenant_name',
  invalid_nit: 'nit',
  invalid_nit_check_digit: 'nit',
  invalid_user_name: 'name',
  invalid_email: 'email',
  password_too_short: 'password',
  password_too_long: 'password',
  unknown_business_template: 'business_template',
};

/** Creates the company and enters the app; a failure goes to its field or to the form. */
export function useSignUp(setError: UseFormSetError<SignUpValues>) {
  const { startSession } = useAuth();
  const { formError, handleError, clearFormError } = useApiFormErrors(setError, fieldByErrorCode);

  async function submit(values: SignUpValues) {
    clearFormError();
    try {
      startSession(await signUp(values));
    } catch (error) {
      handleError(error);
    }
  }

  return { formError, submit };
}

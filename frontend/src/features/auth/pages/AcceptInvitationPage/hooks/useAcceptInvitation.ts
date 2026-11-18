import type { UseFormSetError } from 'react-hook-form';
import { useApiFormErrors, type FieldsByErrorCode } from '@/shared/forms/useApiFormErrors';
import { acceptInvitation } from '../../../api/links';
import type { AcceptInvitationValues } from '../../../model/passwordSchemas';
import { useAuth } from '../../../session/useAuth';

// The field each 422 code of POST /auth/invitations/accept belongs to.
const fieldByErrorCode: FieldsByErrorCode<AcceptInvitationValues> = {
  invalid_user_name: 'name',
  password_too_short: 'password',
  password_too_long: 'password',
  password_too_common: 'password',
};

/** Accepts the invitation and enters the app with its session; GuestOnly then leads in. */
export function useAcceptInvitation(token: string, setError: UseFormSetError<AcceptInvitationValues>) {
  const { startSession } = useAuth();
  const { formError, handleError, clearFormError } = useApiFormErrors(setError, fieldByErrorCode);

  async function submit(values: AcceptInvitationValues) {
    clearFormError();
    try {
      startSession(await acceptInvitation(token, values));
    } catch (error) {
      handleError(error);
    }
  }

  return { formError, submit };
}

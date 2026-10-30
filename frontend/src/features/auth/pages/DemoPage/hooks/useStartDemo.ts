import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useNavigate } from 'react-router';
import { paths } from '@/app/routes/paths';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { startDemo } from '../../../api/demo';
import type { DemoOption } from '../../../api/types';
import type { FromState } from '../../../session/redirect';
import { demoSchema, type DemoValues } from '../../../model/schemas';
import { useAuth } from '../../../session/useAuth';

// No error of POST /auth/demo belongs to a field: the choice is always valid.
const noFields = {};

// Where each role starts: the cashier at the POS, the rest at the start.
const landing = (role: DemoValues['role']) => (role === 'cashier' ? paths.sales.pos() : paths.home());

/**
 * The demo's form: the first business type and the owner are proposed, and a
 * successful entry stores the session and opens the role's first screen.
 */
export function useStartDemo(demos: DemoOption[]) {
  const { startSession } = useAuth();
  const navigate = useNavigate();
  const form = useForm<DemoValues>({
    resolver: zodResolver(demoSchema),
    defaultValues: { business_template: demos[0]?.business_template ?? '', role: 'owner' },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, noFields);

  async function submit(values: DemoValues) {
    clearFormError();
    try {
      const session = await startDemo(values);
      // GuestOnly leads a new session to where the state says it was going.
      const state: FromState = { from: { pathname: landing(values.role), search: '' } };
      // flushSync: the router commits the state before the session re-renders GuestOnly.
      await navigate('.', { replace: true, state, flushSync: true });
      startSession(session);
    } catch (error) {
      handleError(error);
    }
  }

  return { form, formError, submit };
}

import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { startDemo } from '../../../api/demo';
import type { DemoOption } from '../../../api/types';
import { demoSchema, type DemoValues } from '../../../model/schemas';
import { useAuth } from '../../../session/useAuth';

// No error of POST /auth/demo belongs to a field: the choice is always valid.
const noFields = {};

/**
 * The demo's form: the first business type and the owner are proposed, and a
 * successful entry stores the session; GuestOnly then leads to the app.
 */
export function useStartDemo(demos: DemoOption[]) {
  const { startSession } = useAuth();
  const form = useForm<DemoValues>({
    resolver: zodResolver(demoSchema),
    defaultValues: { business_template: demos[0]?.business_template ?? '', role: 'owner' },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(form.setError, noFields);

  async function submit(values: DemoValues) {
    clearFormError();
    try {
      startSession(await startDemo(values));
    } catch (error) {
      handleError(error);
    }
  }

  return { form, formError, submit };
}

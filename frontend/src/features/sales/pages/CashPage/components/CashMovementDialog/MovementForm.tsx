import { zodResolver } from '@hookform/resolvers/zod';
import { useForm } from 'react-hook-form';
import { useApiFormErrors } from '@/shared/forms/useApiFormErrors';
import { Alert, Field, Input } from '@/shared/ui';
import { useRecordCashMovement } from '../../../../api/cashSessions';
import type { CashMovementKind } from '../../../../model/cashLabels';
import { cashMovementSchema, type CashMovementFormInput, type CashMovementValues } from '../../../../model/cashSchemas';
import styles from './CashMovementDialog.module.scss';

// No error of POST /cash-sessions/{id}/movements belongs to a field.
const noFields = {};

type Props = { id: string; sessionId: string; kind: CashMovementKind; reasonHint: string; onDone: () => void };

/** The amount and reason of a movement; the dialog's button submits it. */
export function MovementForm({ id, sessionId, kind, reasonHint, onDone }: Props) {
  const record = useRecordCashMovement(sessionId);
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<CashMovementFormInput, unknown, CashMovementValues>({
    resolver: zodResolver(cashMovementSchema),
    defaultValues: { amount: '', reason: '' },
  });
  const { formError, handleError, clearFormError } = useApiFormErrors(setError, noFields);

  const submit = handleSubmit((values) => {
    clearFormError();
    record.mutate({ kind, ...values }, { onSuccess: onDone, onError: handleError });
  });

  return (
    <form
      id={id}
      className={styles.root}
      noValidate
      onSubmit={(event) => void submit(event)}
      aria-busy={record.isPending || undefined}
    >
      {formError && <Alert tone="danger">{formError}</Alert>}
      <Field label="Valor" error={errors.amount?.message}>
        <Input inputMode="decimal" align="end" autoComplete="off" autoFocus {...register('amount')} />
      </Field>
      <Field label="Motivo" hint={reasonHint} error={errors.reason?.message}>
        <Input autoComplete="off" maxLength={200} {...register('reason')} />
      </Field>
    </form>
  );
}

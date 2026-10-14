import { zodResolver } from '@hookform/resolvers/zod';
import { useId, useState } from 'react';
import { useForm } from 'react-hook-form';
import { z } from 'zod';
import { describeError } from '@/shared/api/errors';
import { Alert, Button, Dialog, DialogClose, Field, Input } from '@/shared/ui';
import { useRecordCashMovement } from './api';
import { moneyField, type CashMovementKind } from './cash';
import styles from './CashMovementDialog.module.scss';

const schema = z.object({
  amount: moneyField('positive', 'Escriba el valor'),
  reason: z.string().trim().min(1, 'Escriba el motivo').max(200, 'Use máximo 200 caracteres'),
});
type Input = z.input<typeof schema>;
type Values = z.output<typeof schema>;

const copy: Record<CashMovementKind, { title: string; description: string; reason: string }> = {
  in: {
    title: 'Registrar ingreso',
    description: 'Efectivo que entra al cajón sin ser una venta, como más sencillo.',
    reason: 'Por ejemplo: sencillo que trajo el dueño',
  },
  out: {
    title: 'Registrar egreso',
    description: 'Efectivo que sale del cajón, como un pago o un retiro.',
    reason: 'Por ejemplo: pago del domicilio',
  },
};

type Props = {
  sessionId: string;
  /** The movement being recorded; without it the dialog is closed. */
  kind?: CashMovementKind;
  onOpenChange: (open: boolean) => void;
};

/** A cash in or out of the open session, with its reason. */
export function CashMovementDialog({ sessionId, kind, onOpenChange }: Props) {
  const text = copy[kind ?? 'in'];
  const formId = useId();
  return (
    <Dialog
      open={kind !== undefined}
      onOpenChange={onOpenChange}
      title={text.title}
      description={text.description}
      footer={
        <>
          <DialogClose asChild>
            <Button variant="secondary">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId}>
            {text.title}
          </Button>
        </>
      }
    >
      {/* Mounted per opening: every movement starts empty. */}
      {kind && <MovementForm id={formId} sessionId={sessionId} kind={kind} onDone={() => onOpenChange(false)} />}
    </Dialog>
  );
}

function MovementForm({
  id,
  sessionId,
  kind,
  onDone,
}: {
  id: string;
  sessionId: string;
  kind: CashMovementKind;
  onDone: () => void;
}) {
  const record = useRecordCashMovement(sessionId);
  const [formError, setFormError] = useState<string>();
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<Input, unknown, Values>({
    resolver: zodResolver(schema),
    defaultValues: { amount: '', reason: '' },
  });

  function submit(values: Values) {
    setFormError(undefined);
    record.mutate(
      { kind, ...values },
      {
        onSuccess: onDone,
        onError: (error) => setFormError(describeError(error).message),
      },
    );
  }

  return (
    <form
      id={id}
      className={styles.root}
      noValidate
      onSubmit={handleSubmit(submit)}
      aria-busy={record.isPending || undefined}
    >
      {formError && <Alert tone="danger">{formError}</Alert>}
      <Field label="Valor" error={errors.amount?.message}>
        <Input inputMode="decimal" align="end" autoComplete="off" autoFocus {...register('amount')} />
      </Field>
      <Field label="Motivo" hint={copy[kind].reason} error={errors.reason?.message}>
        <Input autoComplete="off" maxLength={200} {...register('reason')} />
      </Field>
    </form>
  );
}

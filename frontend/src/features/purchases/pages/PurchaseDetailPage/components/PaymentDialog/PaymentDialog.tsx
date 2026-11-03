import { useId } from 'react';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { useFieldError } from '@/shared/forms/useFieldError';
import { CashOutOption } from '@/features/sales';
import { formatDay, formatMoney } from '@/shared/format';
import { Alert, Button, Dialog, DialogClose, Field, Input, Stack } from '@/shared/ui';
import type { Purchase } from '../../../../api/types';
import { paymentMethodLabels } from '../../../../model/purchaseLabels';
import { usePaymentForm } from '../../hooks/usePaymentForm';

const methods = Object.entries(paymentMethodLabels).map(([value, label]) => ({ value, label }));

type Props = { purchase: Purchase; supplierName: string; onClose: () => void };

function owed(purchase: Purchase): string {
  const due = purchase.due_on ? `, vence el ${formatDay(purchase.due_on)}` : '';
  return `Se deben ${formatMoney(purchase.balance_due)}${due}.`;
}

/**
 * A payment to the supplier of a purchase on credit, with what is owed in
 * view. In cash it can come out of the user's open drawer. Mounted per opening.
 */
export function PaymentDialog({ purchase, supplierName, onClose }: Props) {
  const formId = useId();
  const { form, submit, formError, cashOut, pending } = usePaymentForm(purchase, supplierName, onClose);
  const errorOf = useFieldError(form.control);
  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Registrar un pago a ${supplierName}`}
      description={owed(purchase)}
      footer={
        <>
          <DialogClose asChild>
            <Button variant="secondary">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} disabled={pending} aria-busy={pending || undefined}>
            Registrar pago
          </Button>
        </>
      }
    >
      <form id={formId} noValidate onSubmit={(event) => void submit(event)}>
        <Stack gap="4">
          {formError && <Alert tone="danger">{formError}</Alert>}
          <Field label="Medio de pago" error={errorOf('method')}>
            <SelectController control={form.control} name="method" options={methods} placeholder="Elija el medio" />
          </Field>
          <Field label="Valor" error={errorOf('amount')}>
            <Input inputMode="decimal" align="end" autoComplete="off" {...form.register('amount')} />
          </Field>
          <Field label="Fecha del pago" error={errorOf('paid_on')}>
            <Input type="date" {...form.register('paid_on')} />
          </Field>
          {cashOut.option && <CashOutOption {...cashOut.option} />}
          {cashOut.recordByHand && (
            <Alert tone="info">Si el dinero sale del cajón, regístrelo como egreso en la caja.</Alert>
          )}
        </Stack>
      </form>
    </Dialog>
  );
}

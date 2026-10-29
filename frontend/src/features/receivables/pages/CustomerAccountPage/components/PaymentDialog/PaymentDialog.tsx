import { useId, useState } from 'react';
import { SelectController } from '@/shared/forms/SelectController/SelectController';
import { formatMoney } from '@/shared/format';
import { Alert, Button, Dialog, DialogClose, Field, IconCash, Input } from '@/shared/ui';
import { paymentMethodOptions } from '../../../../model/payment/paymentMethods';
import { usePaymentForm } from '../../hooks/usePaymentForm';
import styles from './PaymentDialog.module.scss';

type Props = { customerId: string; balance: string };

/** Records a partial or total payment; it goes to the receivable due first. */
export function PaymentDialog({ customerId, balance }: Props) {
  const formId = useId();
  const [open, setOpen] = useState(false);
  const payment = usePaymentForm({ customerId, balance, onDone: () => setOpen(false) });
  const { errors } = payment.form.formState;

  function changeOpen(next: boolean) {
    if (next) payment.reset();
    setOpen(next);
  }

  return (
    <Dialog
      open={open}
      onOpenChange={changeOpen}
      trigger={<Button icon={<IconCash />}>Registrar abono</Button>}
      title="Registrar abono"
      description={`Debe ${formatMoney(balance)}. El abono se aplica a la venta que vence primero.`}
      footer={
        <>
          <DialogClose asChild>
            <Button variant="ghost">Cancelar</Button>
          </DialogClose>
          <Button type="submit" form={formId} loading={payment.isSaving}>
            Registrar abono
          </Button>
        </>
      }
    >
      <form id={formId} className={styles.root} noValidate onSubmit={(event) => void payment.submit(event)}>
        {payment.formError && <Alert tone="danger">{payment.formError}</Alert>}
        <Field label="Medio de pago">
          <SelectController control={payment.form.control} name="method" options={paymentMethodOptions} />
        </Field>
        <Field label="Valor del abono" hint="En efectivo entra a su caja." error={errors.amount?.message}>
          <Input align="end" inputMode="decimal" autoComplete="off" autoFocus {...payment.form.register('amount')} />
        </Field>
      </form>
    </Dialog>
  );
}

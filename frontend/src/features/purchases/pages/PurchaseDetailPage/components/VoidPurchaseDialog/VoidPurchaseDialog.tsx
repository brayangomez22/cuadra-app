import { useId } from 'react';
import { Alert, Button, Dialog, DialogClose, Field, Stack, Textarea } from '@/shared/ui';
import type { Purchase } from '../../../../api/types';
import { useVoidPurchaseForm } from '../../hooks/useVoidPurchaseForm';

type Props = { purchase: Purchase; onClose: () => void };

/** Asks why the purchase is voided and says what it does to the stock before doing it. Mounted per opening. */
export function VoidPurchaseDialog({ purchase, onClose }: Props) {
  const formId = useId();
  const { form, submit, formError, pending } = useVoidPurchaseForm(purchase, onClose);
  const lines = purchase.lines.length;
  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Anular la compra ${purchase.invoice_number}`}
      description="La compra queda marcada como anulada y ya no se debe. No se puede deshacer."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="secondary">Conservar compra</Button>
          </DialogClose>
          <Button type="submit" form={formId} variant="danger" disabled={pending} aria-busy={pending || undefined}>
            Anular compra
          </Button>
        </>
      }
    >
      <form id={formId} noValidate onSubmit={(event) => void submit(event)}>
        <Stack gap="4">
          {formError && <Alert tone="danger">{formError}</Alert>}
          <p>
            {`Sale del inventario de la sede lo que entró con sus ${String(lines)} ${lines === 1 ? 'línea' : 'líneas'}. `}
            Si algo ya se vendió o se movió, la compra no se anula.
          </p>
          <Field label="Motivo" error={form.formState.errors.reason?.message}>
            <Textarea autoFocus maxLength={500} rows={3} {...form.register('reason')} />
          </Field>
        </Stack>
      </form>
    </Dialog>
  );
}

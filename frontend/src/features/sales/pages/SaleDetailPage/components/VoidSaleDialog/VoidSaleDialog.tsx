import { useId } from 'react';
import { Alert, Button, Dialog, DialogClose, Field, Textarea } from '@/shared/ui';
import type { Sale } from '../../../../api/types';
import { useVoidSaleForm } from '../../hooks/useVoidSaleForm';
import styles from './VoidSaleDialog.module.scss';

type Props = { sale: Sale; onClose: () => void };

/**
 * Asks why the sale is voided and says what it does with the stock and the
 * money before doing it: the API does not move the drawer. Mounted per opening.
 */
export function VoidSaleDialog({ sale, onClose }: Props) {
  const formId = useId();
  const { form, submit, formError, notices, drawerFailed, canSubmit, pending } = useVoidSaleForm(sale, onClose);
  return (
    <Dialog
      open
      onOpenChange={(open) => !open && onClose()}
      title={`Anular la venta N.º ${sale.number}`}
      description="La venta conserva su número y queda marcada como anulada. No se puede deshacer."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="secondary">Conservar venta</Button>
          </DialogClose>
          <Button type="submit" form={formId} variant="danger" disabled={!canSubmit} aria-busy={pending || undefined}>
            Anular venta
          </Button>
        </>
      }
    >
      <form id={formId} className={styles.root} noValidate onSubmit={(event) => void submit(event)}>
        {formError && <Alert tone="danger">{formError}</Alert>}
        {drawerFailed && (
          <Alert tone="danger">No pudimos revisar la caja de esta venta. Cierre este aviso e intente de nuevo.</Alert>
        )}
        {notices.length > 0 && (
          <ul className={styles.notices} aria-label="Qué pasa al anular">
            {notices.map((notice) => (
              <li key={notice}>{notice}</li>
            ))}
          </ul>
        )}
        <Field label="Motivo" error={form.formState.errors.reason?.message}>
          <Textarea autoFocus maxLength={500} rows={3} {...form.register('reason')} />
        </Field>
      </form>
    </Dialog>
  );
}

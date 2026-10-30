import type { ReactNode } from 'react';
import { formatDay, formatMoney } from '@/shared/format';
import { Button, Dialog, DialogClose } from '@/shared/ui';
import type { PurchaseReview } from '../../../../model/purchaseForm/purchaseReview';
import styles from './ConfirmPurchaseDialog.module.scss';

type Props = {
  review: PurchaseReview | null;
  saving: boolean;
  onConfirm: () => void;
  onCancel: () => void;
  onCloseAutoFocus: (event: Event) => void;
};

/**
 * The last look before registering: a purchase moves stock and costs and
 * cannot be voided yet, so the invoice is read back in one place.
 */
export function ConfirmPurchaseDialog({ review, saving, onConfirm, onCancel, onCloseAutoFocus }: Props) {
  return (
    <Dialog
      open={review !== null}
      onOpenChange={(open) => !open && !saving && onCancel()}
      onCloseAutoFocus={onCloseAutoFocus}
      title="Registrar esta compra"
      description="Al registrarla, la mercancía entra al inventario de la sede y se actualiza el costo de los productos."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="secondary" disabled={saving}>
              Revisar
            </Button>
          </DialogClose>
          <Button loading={saving} onClick={onConfirm}>
            Registrar compra
          </Button>
        </>
      }
    >
      {review && (
        <dl className={styles.root}>
          <Datum term="Proveedor">{review.supplier}</Datum>
          <Datum term="Factura">{review.invoice}</Datum>
          <Datum term="Sede">{review.location}</Datum>
          <Datum term="Productos">{review.lines === 1 ? '1 línea' : `${String(review.lines)} líneas`}</Datum>
          <Datum term="Forma de pago">{payment(review.dueOn)}</Datum>
          <Datum term="Total" emphasis>
            {formatMoney(review.total)}
          </Datum>
        </dl>
      )}
    </Dialog>
  );
}

function payment(dueOn: string | null) {
  if (dueOn === null) return 'Contado';
  return dueOn ? `Crédito, vence el ${formatDay(dueOn)}` : 'Crédito, con el plazo del proveedor';
}

function Datum({ term, emphasis, children }: { term: string; emphasis?: boolean; children: ReactNode }) {
  return (
    <div className={styles.datum} data-emphasis={emphasis ? true : undefined}>
      <dt>{term}</dt>
      <dd>{children}</dd>
    </div>
  );
}

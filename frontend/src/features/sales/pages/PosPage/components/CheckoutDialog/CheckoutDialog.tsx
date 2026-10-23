import { Dialog } from '@/shared/ui';
import type { Paper } from '../../../../Receipt';
import type { Checkout } from '../../hooks/useCheckout';
import { PaymentsForm } from '../PaymentsForm/PaymentsForm';
import { SaleDone } from '../SaleDone/SaleDone';

type Props = {
  checkout: Checkout;
  /** The total of the cart, IVA included. */
  total: string;
  /** The receipt's paper width and how to change it. */
  paper: Paper;
  onPaperChange: (paper: Paper) => void;
};

/** Takes the payments of the sale, mixed if needed, and shows the change. */
export function CheckoutDialog({ checkout, total, paper, onPaperChange }: Props) {
  const { open, sale } = checkout;
  return (
    <Dialog
      open={open}
      onOpenChange={checkout.onOpenChange}
      size="md"
      title={sale ? `Venta N.º ${sale.number}` : 'Cobrar'}
      description={sale ? 'La venta quedó registrada.' : 'Con un solo pago en efectivo vacío se cobra el valor exacto.'}
    >
      {/* Mounted per opening: every charge starts with one empty cash payment. */}
      {open &&
        (sale ? (
          <SaleDone
            sale={sale}
            onNewSale={checkout.close}
            paper={paper}
            onPaperChange={onPaperChange}
            onPrint={() => window.print()}
          />
        ) : (
          <PaymentsForm total={total} onConfirm={checkout.confirm} pending={checkout.pending} error={checkout.error} />
        ))}
    </Dialog>
  );
}

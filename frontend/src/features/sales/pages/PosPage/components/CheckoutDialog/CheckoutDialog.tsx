import { Dialog } from '@/shared/ui';
import type { Paper } from '../../../../model/paper';
import type { Checkout } from '../../hooks/checkout/useCheckout';
import type { SaleCredit } from '../../hooks/checkout/useSaleCredit';
import { CreditAuthorizationForm } from '../CreditAuthorizationForm/CreditAuthorizationForm';
import { PaymentsForm } from '../PaymentsForm/PaymentsForm';
import { SaleDone } from '../SaleDone/SaleDone';

type Props = {
  checkout: Checkout;
  /** The total of the cart, IVA included. */
  total: string;
  /** What the sale's customer can buy on credit; undefined offers no credit. */
  credit: SaleCredit;
  /** The receipt's paper width and how to change it. */
  paper: Paper;
  onPaperChange: (paper: Paper) => void;
};

// The dialog's title and description at each step: payments, authorization and the sale done.
function heading({ sale, authorization }: Checkout) {
  if (sale) return { title: `Venta N.º ${String(sale.number)}`, description: 'La venta quedó registrada.' };
  if (authorization.pending) {
    return {
      title: 'Autorizar venta a crédito',
      description: 'La venta pasa el cupo del cliente. Un administrador puede autorizarla con su correo y contraseña.',
    };
  }
  return { title: 'Cobrar', description: 'Con un solo pago en efectivo vacío se cobra el valor exacto.' };
}

/** Takes the payments of the sale, mixed if needed, and shows the change. */
export function CheckoutDialog({ checkout, total, credit, paper, onPaperChange }: Props) {
  const { open, sale, authorization } = checkout;
  return (
    <Dialog open={open} onOpenChange={checkout.onOpenChange} size="md" {...heading(checkout)}>
      {/* Mounted per opening: every charge starts with one empty cash payment. */}
      {open && sale && (
        <SaleDone
          sale={sale}
          onNewSale={checkout.close}
          paper={paper}
          onPaperChange={onPaperChange}
          onPrint={() => window.print()}
        />
      )}
      {open && !sale && authorization.pending && (
        <CreditAuthorizationForm
          available={authorization.pending.available}
          onAuthorize={authorization.authorize}
          onBack={authorization.cancel}
          pending={checkout.pending}
          error={checkout.error}
        />
      )}
      {/* Hidden, not unmounted, while authorizing: going back keeps the payments typed. */}
      {open && !sale && (
        <div hidden={Boolean(authorization.pending)}>
          <PaymentsForm
            total={total}
            credit={credit}
            onConfirm={checkout.confirm}
            pending={checkout.pending}
            error={checkout.error}
          />
        </div>
      )}
    </Dialog>
  );
}

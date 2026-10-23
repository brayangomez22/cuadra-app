import { useState } from 'react';
import type { Me } from '@/features/auth';
import type { Location } from '@/features/inventory';
import { loadPaper, savePaper } from '../../../../model/storage';
import { PrintableReceipt } from '../../../../PrintableReceipt';
import type { Checkout } from '../../hooks/useCheckout';
import { CheckoutDialog } from '../CheckoutDialog/CheckoutDialog';

type Props = { checkout: Checkout; total: string; locations: Location[]; user: Me | undefined };

/** The checkout and, once the sale is confirmed, its receipt to print on this device's paper. */
export function SaleCheckout({ checkout, total, locations, user }: Props) {
  const [paper, setPaper] = useState(loadPaper);
  const { sale } = checkout;
  return (
    <>
      <CheckoutDialog
        checkout={checkout}
        total={total}
        paper={paper}
        onPaperChange={(value) => {
          setPaper(value);
          savePaper(value);
        }}
      />
      {sale && user && (
        <PrintableReceipt
          sale={sale}
          business={{ name: user.tenant_name, nit: user.tenant_nit }}
          location={locations.find((l) => l.id === sale.location_id)?.name ?? ''}
          cashier={user.name}
          paper={paper}
        />
      )}
    </>
  );
}

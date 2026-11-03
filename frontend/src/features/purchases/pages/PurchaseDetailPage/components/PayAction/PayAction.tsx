import { useState } from 'react';
import { useCan } from '@/features/auth';
import { Button } from '@/shared/ui';
import { useSupplier } from '../../../../api/suppliers';
import type { Purchase } from '../../../../api/types';
import { canPay } from '../../../../model/purchaseStatus/purchaseStatus';
import { PaymentDialog } from '../PaymentDialog/PaymentDialog';

/** Pay the supplier, for who registers purchases and only while something is owed. */
export function PayAction({ purchase }: { purchase: Purchase }) {
  const canWrite = useCan()('purchases:write');
  const supplier = useSupplier(purchase.supplier_id);
  const [open, setOpen] = useState(false);
  if (!canWrite || !canPay(purchase)) return null;
  return (
    <>
      <Button onClick={() => setOpen(true)}>Registrar pago</Button>
      {open && (
        <PaymentDialog
          purchase={purchase}
          supplierName={supplier.data?.name ?? 'el proveedor'}
          onClose={() => setOpen(false)}
        />
      )}
    </>
  );
}

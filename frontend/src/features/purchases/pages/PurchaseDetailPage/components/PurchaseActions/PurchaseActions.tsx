import { useState } from 'react';
import { useCan } from '@/features/auth';
import { Button } from '@/shared/ui';
import { useSupplier } from '../../../../api/suppliers';
import type { Purchase } from '../../../../api/types';
import { canPay, canVoid } from '../../../../model/purchaseStatus/purchaseStatus';
import { PaymentDialog } from '../PaymentDialog/PaymentDialog';
import { VoidPurchaseDialog } from '../VoidPurchaseDialog/VoidPurchaseDialog';

/** Pay the supplier or void the purchase, for who registers purchases and only when the purchase allows it. */
export function PurchaseActions({ purchase }: { purchase: Purchase }) {
  const canWrite = useCan()('purchases:write');
  const supplier = useSupplier(purchase.supplier_id);
  const [open, setOpen] = useState<'pay' | 'void' | null>(null);
  if (!canWrite) return null;
  const close = () => {
    setOpen(null);
  };
  return (
    <>
      {canPay(purchase) && <Button onClick={() => setOpen('pay')}>Registrar pago</Button>}
      {canVoid(purchase) && (
        <Button variant="danger" onClick={() => setOpen('void')}>
          Anular compra
        </Button>
      )}
      {open === 'pay' && (
        <PaymentDialog purchase={purchase} supplierName={supplier.data?.name ?? 'el proveedor'} onClose={close} />
      )}
      {open === 'void' && <VoidPurchaseDialog purchase={purchase} onClose={close} />}
    </>
  );
}

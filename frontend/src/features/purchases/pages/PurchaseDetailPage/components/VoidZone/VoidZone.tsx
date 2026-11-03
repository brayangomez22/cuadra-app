import { useState } from 'react';
import { useCan } from '@/features/auth';
import { Button } from '@/shared/ui';
import type { Purchase } from '../../../../api/types';
import { canVoid } from '../../../../model/purchaseStatus/purchaseStatus';
import { VoidPurchaseDialog } from '../VoidPurchaseDialog/VoidPurchaseDialog';
import styles from './VoidZone.module.scss';

/**
 * Voiding, at the end of the page and apart from paying: it cannot be undone,
 * and next to the most frequent action a slip of the thumb would open it.
 */
export function VoidZone({ purchase }: { purchase: Purchase }) {
  const canWrite = useCan()('purchases:write');
  const [open, setOpen] = useState(false);
  if (!canWrite || !canVoid(purchase)) return null;
  return (
    <section className={styles.root} aria-label="Anular la compra">
      <p className={styles.text}>
        ¿La compra se registró mal? Anularla saca del inventario lo que entró y deja de deberse.
      </p>
      <Button variant="danger" onClick={() => setOpen(true)}>
        Anular compra
      </Button>
      {open && <VoidPurchaseDialog purchase={purchase} onClose={() => setOpen(false)} />}
    </section>
  );
}

import { formatMoney, isZero } from '@/shared/format';
import { Button, DialogClose } from '@/shared/ui';
import type { Settlement } from '../../../../model/cart/settlement';
import styles from './CheckoutFooter.module.scss';

type Props = { settlement: Settlement; pending: boolean };

/**
 * CheckoutFooter stays at the bottom of the checkout while the payments
 * scroll: what was received, the change or what is missing, and the
 * buttons, so the cashier reads the figures right where they confirm.
 */
export function CheckoutFooter({ settlement, pending }: Props) {
  const missing = !isZero(settlement.missing);
  const change = !missing && !isZero(settlement.change);
  return (
    <div role="group" aria-label="Cuenta del cobro" className={styles.root}>
      <dl className={styles.figures}>
        <div className={styles.figure}>
          <dt>Recibido</dt>
          <dd>{formatMoney(settlement.paid)}</dd>
        </div>
        <div className={styles.figure} data-tone={missing ? 'missing' : change ? 'change' : undefined}>
          <dt>{missing ? 'Falta' : 'Cambio'}</dt>
          <dd className={styles.due}>{formatMoney(missing ? settlement.missing : settlement.change)}</dd>
        </div>
      </dl>
      <div className={styles.actions}>
        <DialogClose asChild>
          <Button variant="secondary">Volver</Button>
        </DialogClose>
        <Button type="submit" size="lg" loading={pending}>
          Confirmar venta
        </Button>
      </div>
    </div>
  );
}

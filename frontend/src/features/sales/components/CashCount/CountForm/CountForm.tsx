import { formatMoney } from '@/shared/format';
import { Field, Textarea } from '@/shared/ui';
import type { CashSession } from '../../../api/types';
import { useCashCount } from '../../../hooks/useCashCount';
import { ConfirmCloseDialog } from '../ConfirmCloseDialog/ConfirmCloseDialog';
import { CountSummary } from '../CountSummary/CountSummary';
import { DenominationCounts } from '../DenominationCounts/DenominationCounts';
import styles from './CountForm.module.scss';

type Props = {
  session: CashSession;
  /** Whose drawer it is, when it is another user's: the confirmation names them. */
  cashier?: string;
  notesHint?: string;
  onClosed: (id: string) => void;
};

/**
 * The count by denomination with its notes, the total counted and the
 * confirmation to close. It says to count the whole drawer, opening float
 * included: a cashier who sets the float aside would show it as a shortage.
 */
export function CountForm({
  session,
  cashier,
  notesHint = 'Lo que el dueño deba saber de este cierre.',
  onClosed,
}: Props) {
  const count = useCashCount(session, onClosed);

  return (
    <form
      className={styles.root}
      noValidate
      onSubmit={(event) => {
        event.preventDefault();
        count.submit();
      }}
    >
      <div className={styles.count}>
        <p className={styles.instructions}>
          <strong>
            Cuente todo el efectivo del cajón, incluida la base de {formatMoney(session.opening_float)} con la que abrió
            {cashier ? ` ${cashier}` : ''}.
          </strong>{' '}
          No reste nada: Cuadra hace las cuentas al cerrar. Los váuchers de tarjeta y los comprobantes de transferencias
          no se cuentan.
        </p>
        <DenominationCounts
          texts={count.texts}
          invalid={count.submitted ? count.invalid : undefined}
          onChange={count.setCount}
        />
        <Field label="Notas" optional hint={notesHint}>
          <Textarea maxLength={500} value={count.notes} onChange={(event) => count.setNotes(event.target.value)} />
        </Field>
      </div>

      <CountSummary total={count.total} />

      <ConfirmCloseDialog
        open={count.confirming}
        onOpenChange={count.setConfirming}
        total={count.total}
        cashier={cashier}
        onConfirm={count.confirm}
        closing={count.close}
      />
    </form>
  );
}

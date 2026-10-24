import { Field, Textarea } from '@/shared/ui';
import type { CashSession } from '../../../../api/types';
import { useCashCount } from '../../hooks/useCashCount';
import { ConfirmCloseDialog } from '../ConfirmCloseDialog/ConfirmCloseDialog';
import { CountSummary } from '../CountSummary/CountSummary';
import { DenominationCounts } from '../DenominationCounts/DenominationCounts';
import styles from './CountForm.module.scss';

type Props = { session: CashSession; onClosed: (id: string) => void };

/** The count by denomination with its notes, the total counted and the confirmation to close. */
export function CountForm({ session, onClosed }: Props) {
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
        <DenominationCounts
          texts={count.texts}
          invalid={count.submitted ? count.invalid : undefined}
          onChange={count.setCount}
        />
        <Field label="Notas" optional hint="Lo que el dueño deba saber de este cierre.">
          <Textarea maxLength={500} value={count.notes} onChange={(event) => count.setNotes(event.target.value)} />
        </Field>
      </div>

      <CountSummary total={count.total} />

      <ConfirmCloseDialog
        open={count.confirming}
        onOpenChange={count.setConfirming}
        total={count.total}
        onConfirm={count.confirm}
        closing={count.close}
      />
    </form>
  );
}

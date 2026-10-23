import { Field, IconButton, IconClose, Input, Select } from '@/shared/ui';
import { methodLabels, type PaymentMethod } from '../../../../model/paymentMethods';
import type { PaymentRow as Row } from '../../../../model/payments';
import styles from './PaymentRow.module.scss';

type Props = {
  row: Row;
  /** Its place among the payments, from 0: the first one has the focus and cannot be removed. */
  index: number;
  /** What it shows when the amount is empty, and its error once submitted. */
  field: { placeholder?: string; error?: string };
  onChange: (changes: Partial<Row>) => void;
  onRemove: () => void;
};

const methodOptions = Object.entries(methodLabels).map(([value, label]) => ({ value, label }));

/** A payment of the sale: its method and its amount. */
export function PaymentRow({ row, index, field, onChange, onRemove }: Props) {
  const n = index + 1;
  return (
    <div className={styles.root}>
      <Field
        className={styles.method}
        label={
          <>
            Medio de pago <span className="srOnly">{n}</span>
          </>
        }
      >
        <Select
          options={methodOptions}
          value={row.method}
          onValueChange={(value) => onChange({ method: value as PaymentMethod })}
        />
      </Field>
      <Field
        className={styles.amount}
        label={
          <>
            Valor <span className="srOnly">del pago {n}</span>
          </>
        }
        error={field.error}
      >
        <Input
          align="end"
          inputMode="decimal"
          autoComplete="off"
          placeholder={field.placeholder}
          value={row.text}
          onChange={(event) => onChange({ text: event.target.value })}
          autoFocus={index === 0}
        />
      </Field>
      {index > 0 && (
        <IconButton className={styles.remove} label={`Quitar pago ${n}`} icon={<IconClose />} onClick={onRemove} />
      )}
    </div>
  );
}

import { formatMoney } from '@/shared/format';
import { Field, Input } from '@/shared/ui';
import {
  denominationPrefix,
  denominationSubtotal,
  parseCount,
  type Denomination,
} from '../../../model/cash/denominations';
import styles from './DenominationField.module.scss';

type Props = {
  denomination: Denomination;
  text: string;
  invalid: boolean;
  onChange: (text: string) => void;
};

/**
 * DenominationField is how many bills or coins of one denomination, with what
 * they add up to under it. The label shows the amount; "Billetes de" is for
 * screen readers, since the group's title says it on screen.
 */
export function DenominationField({ denomination, text, invalid, onChange }: Props) {
  const parsed = parseCount(text);
  const subtotal = parsed.ok
    ? denominationSubtotal({ denomination: denomination.value, quantity: parsed.quantity })
    : undefined;
  return (
    <div className={styles.root}>
      <Field
        label={
          <>
            <span className="srOnly">{denominationPrefix(denomination)}</span>
            {formatMoney(denomination.value)}
          </>
        }
        error={invalid ? 'Escriba cuántos hay, sin decimales' : undefined}
      >
        <Input
          inputMode="numeric"
          align="end"
          autoComplete="off"
          value={text}
          onChange={(event) => onChange(event.target.value)}
        />
      </Field>
      <span className={styles.subtotal} aria-hidden="true">
        {subtotal && subtotal !== '0' ? `= ${formatMoney(subtotal)}` : ''}
      </span>
    </div>
  );
}

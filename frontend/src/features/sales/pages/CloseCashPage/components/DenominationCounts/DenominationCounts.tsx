import { formatMoney } from '@/shared/format';
import { Field, Input } from '@/shared/ui';
import { denominationLabel, denominations, denominationSubtotal, parseCount } from '../../../../model/denominations';
import styles from './DenominationCounts.module.scss';

type Props = {
  /** What was typed per denomination. */
  texts: Record<string, string>;
  /** The denominations to mark as wrong; undefined until the count is submitted. */
  invalid?: Set<string>;
  onChange: (denomination: string, text: string) => void;
};

/** A field per bill and coin, with what it adds up to beside it. */
export function DenominationCounts({ texts, invalid, onChange }: Props) {
  return (
    <fieldset className={styles.root}>
      <legend className="srOnly">Conteo por denominación</legend>
      {denominations.map((d) => {
        const parsed = parseCount(texts[d.value] ?? '');
        const subtotal = parsed.ok
          ? denominationSubtotal({ denomination: d.value, quantity: parsed.quantity })
          : undefined;
        return (
          <div key={d.value} className={styles.denomination}>
            <Field
              label={denominationLabel(d)}
              error={invalid?.has(d.value) ? 'Escriba cuántos hay, sin decimales' : undefined}
            >
              <Input
                inputMode="numeric"
                align="end"
                autoComplete="off"
                value={texts[d.value] ?? ''}
                onChange={(event) => onChange(d.value, event.target.value)}
              />
            </Field>
            <span className={styles.subtotal} aria-hidden="true">
              {subtotal && subtotal !== '0' ? formatMoney(subtotal) : ''}
            </span>
          </div>
        );
      })}
    </fieldset>
  );
}

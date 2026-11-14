import { denominationGroups } from '../../../model/cash/denominations';
import { DenominationField } from '../DenominationField/DenominationField';
import styles from './DenominationCounts.module.scss';

type Props = {
  /** What was typed per denomination. */
  texts: Record<string, string>;
  /** The denominations to mark as wrong; undefined until the count is submitted. */
  invalid?: Set<string>;
  onChange: (denomination: string, text: string) => void;
};

/** The bills, then the coins: a field per denomination, up to three per line. */
export function DenominationCounts({ texts, invalid, onChange }: Props) {
  return (
    <fieldset className={styles.root}>
      <legend className="srOnly">Conteo por denominación</legend>
      {denominationGroups.map((group) => (
        <fieldset key={group.title} className={styles.group}>
          <legend className={styles.title}>{group.title}</legend>
          {group.note && <p className={styles.note}>{group.note}</p>}
          <div className={styles.fields}>
            {group.denominations.map((d) => (
              <DenominationField
                key={d.value}
                denomination={d}
                text={texts[d.value] ?? ''}
                invalid={invalid?.has(d.value) ?? false}
                onChange={(text) => onChange(d.value, text)}
              />
            ))}
          </div>
        </fieldset>
      ))}
    </fieldset>
  );
}

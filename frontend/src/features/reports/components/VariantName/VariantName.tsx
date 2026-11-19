import styles from './VariantName.module.scss';

/** A variant's axis values (Rojo · M) under its product's row: indented, the detail of the row above. */
export function VariantName({ label }: { label: string }) {
  return <span className={styles.root}>{label}</span>;
}

import clsx from 'clsx';
import styles from './Spinner.module.scss';

type Props = {
  /** What is loading, read by screen readers. */
  label?: string;
  /** Inside a control that already announces it (aria-busy): no status role. */
  decorative?: boolean;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
};

export function Spinner({ label = 'Cargando…', size = 'md', decorative = false, className }: Props) {
  if (decorative) {
    return (
      <span className={clsx(styles.root, className)} data-size={size} aria-hidden="true">
        <span className={styles.ring} />
      </span>
    );
  }
  return (
    <span role="status" className={clsx(styles.root, className)} data-size={size}>
      <span className={styles.ring} aria-hidden="true" />
      <span className="srOnly">{label}</span>
    </span>
  );
}

import clsx from 'clsx';
import styles from './Brand.module.scss';

/** Brand: the Cuadra wordmark with its orange bar, in the top bar and the side menu. */
export function Brand({ className }: { className?: string }) {
  return <span className={clsx(styles.root, className)}>Cuadra</span>;
}

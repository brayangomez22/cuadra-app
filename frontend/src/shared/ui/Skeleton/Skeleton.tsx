import clsx from 'clsx';
import styles from './Skeleton.module.scss';

type Props = {
  /** text: one line at the current font size. block: fills its container. */
  shape?: 'text' | 'block';
  className?: string;
};

// Decorative placeholder. The loading state is announced by whoever renders it
// (aria-busy on the region, or a Spinner).
export function Skeleton({ shape = 'text', className }: Props) {
  return <span className={clsx(styles.root, className)} data-shape={shape} aria-hidden="true" />;
}

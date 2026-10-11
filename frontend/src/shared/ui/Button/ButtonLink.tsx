import clsx from 'clsx';
import type { ReactNode } from 'react';
import { Link, type LinkProps } from 'react-router';
import styles from './Button.module.scss';

type Props = LinkProps & {
  variant?: 'primary' | 'secondary' | 'ghost';
  size?: 'md' | 'lg';
  /** Decorative icon before the label. */
  icon?: ReactNode;
};

/** A link that looks like a Button: it navigates, so it is an <a>, not a <button>. */
export function ButtonLink({ variant = 'primary', size = 'md', icon, className, children, ...rest }: Props) {
  return (
    <Link className={clsx(styles.root, className)} data-variant={variant} data-size={size} {...rest}>
      {icon && <span className={styles.icon}>{icon}</span>}
      <span className={styles.label}>{children}</span>
    </Link>
  );
}

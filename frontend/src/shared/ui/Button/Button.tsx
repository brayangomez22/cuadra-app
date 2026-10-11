import clsx from 'clsx';
import type { ComponentProps, ReactNode } from 'react';
import { Spinner } from '../Spinner';
import styles from './Button.module.scss';

export type ButtonProps = ComponentProps<'button'> & {
  /** primary is the one orange action of the screen (docs/estilos.md). */
  variant?: 'primary' | 'secondary' | 'danger' | 'ghost';
  size?: 'md' | 'lg';
  /** Blocks the button and shows a spinner while the action runs. */
  loading?: boolean;
  /** Decorative icon before the label. */
  icon?: ReactNode;
};

export function Button({
  variant = 'primary',
  size = 'md',
  loading = false,
  icon,
  disabled,
  type = 'button',
  className,
  children,
  ...rest
}: ButtonProps) {
  return (
    <button
      type={type}
      className={clsx(styles.root, className)}
      data-variant={variant}
      data-size={size}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      {...rest}
    >
      {loading ? <Spinner size="sm" decorative className={styles.icon} /> : icon && <span className={styles.icon}>{icon}</span>}
      <span className={styles.label}>{children}</span>
    </button>
  );
}

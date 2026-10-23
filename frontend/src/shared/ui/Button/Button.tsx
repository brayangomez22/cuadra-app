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
  disabled = false,
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
      aria-busy={loading ? true : undefined}
      {...rest}
    >
      <ButtonIcon loading={loading} icon={icon} />
      <span className={styles.label}>{children}</span>
    </button>
  );
}

// While loading, the spinner takes the place of the icon.
function ButtonIcon({ loading, icon }: { loading: boolean; icon: ReactNode }) {
  if (loading) return <Spinner size="sm" decorative className={styles.icon} />;
  return icon && <span className={styles.icon}>{icon}</span>;
}

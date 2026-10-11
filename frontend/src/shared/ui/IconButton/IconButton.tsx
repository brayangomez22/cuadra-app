import clsx from 'clsx';
import type { ComponentProps, ReactNode } from 'react';
import styles from './IconButton.module.scss';

type Props = Omit<ComponentProps<'button'>, 'children' | 'aria-label'> & {
  /** Required: an icon alone has no accessible name. It is also the tooltip. */
  label: string;
  icon: ReactNode;
  variant?: 'ghost' | 'secondary' | 'danger';
};

export function IconButton({ label, icon, variant = 'ghost', type = 'button', className, title, ...rest }: Props) {
  return (
    <button
      type={type}
      className={clsx(styles.root, className)}
      data-variant={variant}
      aria-label={label}
      title={title ?? label}
      {...rest}
    >
      {icon}
    </button>
  );
}

import clsx from 'clsx';
import type { ReactNode } from 'react';
import { IconAlert, IconInfo, IconSuccess } from '../icons';
import styles from './Alert.module.scss';

export type AlertTone = 'danger' | 'warning' | 'info' | 'success';

type Props = {
  tone: AlertTone;
  title?: ReactNode;
  /** What happened and what to do, in plain words. */
  children: ReactNode;
  className?: string;
};

const icons: Record<AlertTone, ReactNode> = {
  danger: <IconAlert />,
  warning: <IconAlert />,
  info: <IconInfo />,
  success: <IconSuccess />,
};

// Alert is an inline message inside the page, such as a form error that is not
// about one field. Errors interrupt the screen reader (role="alert"); the other
// tones wait for a pause (role="status"). Always icon and text, never color alone.
export function Alert({ tone, title, children, className }: Props) {
  return (
    <div className={clsx(styles.root, className)} data-tone={tone} role={tone === 'danger' ? 'alert' : 'status'}>
      <span className={styles.icon}>{icons[tone]}</span>
      <div className={styles.content}>
        {title && <p className={styles.title}>{title}</p>}
        <div>{children}</div>
      </div>
    </div>
  );
}

import type { ReactNode } from 'react';
import { createPortal } from 'react-dom';
import styles from './Printable.module.scss';

/**
 * What the browser prints, in a portal on <body>: while it is mounted,
 * printing (a button or Ctrl+P) shows only it (styles/base/_print). Hidden on
 * screen, and from screen readers, which read the same content on the page.
 */
export function Printable({ children }: { children: ReactNode }) {
  return createPortal(
    <div className={styles.root} data-print-root aria-hidden="true">
      {children}
    </div>,
    document.body,
  );
}

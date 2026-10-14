import { createPortal } from 'react-dom';
import { Receipt } from './Receipt';
import styles from './PrintableReceipt.module.scss';

/**
 * The receipt that the browser prints, in a portal on <body>: while it is
 * mounted, printing (the button or Ctrl+P) shows only it (styles/base/_print).
 * Hidden on screen, and from screen readers, which read the sale in the dialog.
 */
export function PrintableReceipt(props: Parameters<typeof Receipt>[0]) {
  return createPortal(
    <div className={styles.root} data-print-root aria-hidden="true">
      <Receipt {...props} />
    </div>,
    document.body,
  );
}

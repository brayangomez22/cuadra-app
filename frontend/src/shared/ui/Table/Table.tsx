import clsx from 'clsx';
import type { ComponentProps, ReactNode } from 'react';
import styles from './Table.module.scss';

type TableProps = ComponentProps<'table'> & {
  /** Accessible name. Hidden on screen unless showCaption. */
  caption: ReactNode;
  showCaption?: boolean;
};

// Only the table scrolls sideways, inside its own container, never the page.
// The container is focusable so keyboard users can scroll it too.
export function Table({ caption, showCaption = false, className, children, ...rest }: TableProps) {
  return (
    <div
      className={clsx(styles.root, className)}
      tabIndex={0}
      role="group"
      aria-label={typeof caption === 'string' ? caption : undefined}
    >
      <table className={styles.table} {...rest}>
        <caption className={showCaption ? styles.caption : 'srOnly'}>{caption}</caption>
        {children}
      </table>
    </div>
  );
}

export function TableHead({ className, ...rest }: ComponentProps<'thead'>) {
  return <thead className={clsx(styles.head, className)} {...rest} />;
}

export function TableBody({ className, ...rest }: ComponentProps<'tbody'>) {
  return <tbody className={clsx(styles.body, className)} {...rest} />;
}

type RowProps = ComponentProps<'tr'> & { selected?: boolean };

export function TableRow({ selected, className, ...rest }: RowProps) {
  return <tr className={clsx(styles.row, className)} aria-selected={selected ? true : undefined} {...rest} />;
}

/**
 * numeric: prices and quantities, right-aligned with tabular figures. wide: a
 * column that only shows from a tablet up (md), or from lg for a table of many
 * figures; below it a TableNarrow in another cell says what it does, so the
 * table fits without scrolling sideways.
 */
type CellProps<T> = T & { numeric?: boolean; wide?: boolean | 'lg' };

const cellData = (numeric?: boolean, wide?: boolean | 'lg') => ({
  'data-numeric': numeric ? true : undefined,
  'data-wide': wide === false ? undefined : wide,
});

export function TableHeaderCell({ numeric, wide, scope = 'col', className, ...rest }: CellProps<ComponentProps<'th'>>) {
  return <th scope={scope} className={clsx(styles.headerCell, className)} {...cellData(numeric, wide)} {...rest} />;
}

export function TableCell({ numeric, wide, className, ...rest }: CellProps<ComponentProps<'td'>>) {
  return <td className={clsx(styles.cell, className)} {...cellData(numeric, wide)} {...rest} />;
}

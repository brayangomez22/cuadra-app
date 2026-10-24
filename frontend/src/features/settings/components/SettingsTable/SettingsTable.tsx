import type { ReactNode } from 'react';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import { RowActions } from '../RowActions/RowActions';
import styles from './SettingsTable.module.scss';

export type Column<T> = {
  header: string;
  cell: (item: T) => ReactNode;
  /** A code or key, typed exactly as it shows. */
  monospace?: boolean;
};

type Props<T> = {
  caption: string;
  columns: Column<T>[];
  items: T[];
  /** What a row is, for its buttons: "Editar Metro". */
  nameOf: (item: T) => string;
  onEdit: (item: T) => void;
  onDelete: (item: T) => void;
};

/** A settings list: the section's columns, and edit and delete at the end of each row. */
export function SettingsTable<T extends { id: string }>({
  caption,
  columns,
  items,
  nameOf,
  onEdit,
  onDelete,
}: Props<T>) {
  return (
    <Table caption={caption}>
      <TableHead>
        <TableRow>
          {columns.map((column) => (
            <TableHeaderCell key={column.header}>{column.header}</TableHeaderCell>
          ))}
          <TableHeaderCell>
            <span className="srOnly">Acciones</span>
          </TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {items.map((item) => (
          <TableRow key={item.id}>
            {columns.map((column) => (
              <TableCell key={column.header} className={column.monospace ? styles.code : undefined}>
                {column.cell(item)}
              </TableCell>
            ))}
            <TableCell>
              <RowActions name={nameOf(item)} onEdit={() => onEdit(item)} onDelete={() => onDelete(item)} />
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

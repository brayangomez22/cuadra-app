import type { FieldArrayWithId } from 'react-hook-form';
import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { ProductFormInput, VariantRow } from '../../../../model/productSchema';
import { VariantMatrixRow } from '../VariantMatrixRow/VariantMatrixRow';
import styles from './VariantMatrix.module.scss';

type Props = {
  fields: FieldArrayWithId<ProductFormInput, 'variants'>[];
  /** The rows as typed now: a variant saved in an earlier attempt has its id. */
  rows: VariantRow[];
  inactive: boolean;
};

/** The table of the variants: one row per combination, with its own codes and amounts. */
export function VariantMatrix({ fields, rows, inactive }: Props) {
  return (
    <div className={styles.root}>
      <Table caption="Variantes del producto">
        <TableHead>
          <TableRow>
            <TableHeaderCell>Variante</TableHeaderCell>
            <TableHeaderCell>SKU</TableHeaderCell>
            <TableHeaderCell>Código de barras</TableHeaderCell>
            <TableHeaderCell numeric>Costo</TableHeaderCell>
            <TableHeaderCell numeric>Precio (sin IVA)</TableHeaderCell>
            <TableHeaderCell>Activa</TableHeaderCell>
          </TableRow>
        </TableHead>
        <TableBody>
          {fields.map((field, index) => (
            <VariantMatrixRow key={field.id} index={index} row={rows[index] ?? field} inactive={inactive} />
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

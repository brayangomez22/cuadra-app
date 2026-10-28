import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { ImportProductPreview } from '../../../../api/types';
import { sampleRows } from '../../../../model/sampleRows';

/** The first products of the file as the import read them, one row per variant. */
export function SampleTable({ sample }: { sample: ImportProductPreview[] }) {
  return (
    <Table caption="Primeros productos del archivo">
      <TableHead>
        <TableRow>
          <TableHeaderCell numeric>Fila</TableHeaderCell>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell>SKU</TableHeaderCell>
          <TableHeaderCell>Variante</TableHeaderCell>
          <TableHeaderCell numeric>Precio</TableHeaderCell>
          <TableHeaderCell numeric>Stock inicial</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {sampleRows(sample).map((row) => (
          <TableRow key={row.row}>
            <TableCell numeric>{row.row}</TableCell>
            <TableCell>{row.name}</TableCell>
            <TableCell>{row.sku}</TableCell>
            <TableCell>{row.variant}</TableCell>
            <TableCell numeric>{row.price}</TableCell>
            <TableCell numeric>{row.stock}</TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
}

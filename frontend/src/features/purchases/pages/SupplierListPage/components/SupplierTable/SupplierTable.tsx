import { Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import type { Supplier } from '../../../../api/types';
import { SupplierRow } from './SupplierRow';

type Props = { suppliers: Supplier[]; canWrite: boolean };

/** The suppliers of a page: their NIT, term, contact and what is owed to them; each links to its purchases. */
export function SupplierTable({ suppliers, canWrite }: Props) {
  return (
    <Table caption="Proveedores">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Proveedor</TableHeaderCell>
          <TableHeaderCell>NIT</TableHeaderCell>
          <TableHeaderCell>Plazo</TableHeaderCell>
          <TableHeaderCell>Contacto</TableHeaderCell>
          <TableHeaderCell numeric>Se le debe</TableHeaderCell>
          <TableHeaderCell>
            <span className="srOnly">Compras</span>
          </TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {suppliers.map((supplier) => (
          <SupplierRow key={supplier.id} supplier={supplier} canWrite={canWrite} />
        ))}
      </TableBody>
    </Table>
  );
}

import { useParams } from 'react-router';
import { QueryState } from '@/shared/query';
import { Page, Spinner } from '@/shared/ui';
import { useSupplier } from '../../api/suppliers';
import { SupplierForm } from './components/SupplierForm/SupplierForm';

/** Creates a supplier at /compras/proveedores/nuevo and edits one at /compras/proveedores/:supplierId/editar. */
export function SupplierFormPage() {
  const { supplierId } = useParams();
  const supplier = useSupplier(supplierId);
  // key: a different supplier starts a fresh form.
  if (!supplierId || supplier.data) return <SupplierForm key={supplierId ?? 'new'} supplier={supplier.data} />;
  return (
    <Page title="Editar proveedor">
      <QueryState
        queries={[supplier]}
        errorTitle="No pudimos abrir el proveedor"
        loading={<Spinner label="Cargando el proveedor…" />}
      >
        {null}
      </QueryState>
    </Page>
  );
}

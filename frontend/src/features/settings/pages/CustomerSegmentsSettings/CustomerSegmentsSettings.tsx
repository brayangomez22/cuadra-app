import { useCustomerSegments, type CustomerSegment } from '@/features/customers';
import { QueryState } from '@/shared/query';
import { Button, IconPlus, Spinner } from '@/shared/ui';
import { useDeleteCustomerSegment } from '../../api/customerSegments';
import { ConfirmDeleteDialog } from '../../components/ConfirmDeleteDialog/ConfirmDeleteDialog';
import { SettingsSection } from '../../components/SettingsSection/SettingsSection';
import { SettingsTable, type Column } from '../../components/SettingsTable/SettingsTable';
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation';
import { useEditDialog } from '../../hooks/useEditDialog';
import { SegmentDialog } from './components/SegmentDialog/SegmentDialog';
import styles from './CustomerSegmentsSettings.module.scss';

const columns: Column<CustomerSegment>[] = [{ header: 'Segmento', cell: (segment) => segment.name }];

/** /configuracion/segmentos: how the business groups its customers (wholesale, an agreement…). */
export function CustomerSegmentsSettings() {
  const segments = useCustomerSegments();
  const dialog = useEditDialog<CustomerSegment>();
  const deletion = useDeleteConfirmation(useDeleteCustomerSegment().mutateAsync);

  return (
    <SettingsSection
      title="Segmentos de cliente"
      description="Grupos de clientes de su negocio, para clasificarlos y, más adelante, darles precios propios."
      action={
        <Button icon={<IconPlus />} onClick={dialog.openNew}>
          Agregar segmento
        </Button>
      }
    >
      <QueryState
        queries={[segments]}
        errorTitle="No pudimos cargar los segmentos"
        loading={<Spinner label="Cargando segmentos…" />}
        isEmpty={segments.data?.length === 0}
        empty={<p className={styles.empty}>Todavía no hay segmentos. Un cliente puede quedar sin segmento.</p>}
      >
        {() => (
          <SettingsTable
            caption="Segmentos de cliente"
            columns={columns}
            items={segments.data ?? []}
            nameOf={(segment) => segment.name}
            onEdit={dialog.openEdit}
            onDelete={deletion.ask}
          />
        )}
      </QueryState>
      <SegmentDialog open={dialog.isOpen} segment={dialog.editing} onClose={dialog.close} />
      <ConfirmDeleteDialog
        {...deletion.dialogProps}
        title={`¿Borrar el segmento ${deletion.target?.name ?? ''}?`}
        description="Un segmento con clientes, aunque estén inactivos, no se puede borrar."
      />
    </SettingsSection>
  );
}

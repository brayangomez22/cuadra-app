import { useVariantAxes, type VariantAxis } from '@/features/catalog';
import { QueryState } from '@/shared/query';
import { Button, IconPlus, Spinner } from '@/shared/ui';
import { useDeleteVariantAxis } from '../../api/variantAxes';
import { ConfirmDeleteDialog } from '../../components/ConfirmDeleteDialog/ConfirmDeleteDialog';
import { SettingsSection } from '../../components/SettingsSection/SettingsSection';
import { SettingsTable, type Column } from '../../components/SettingsTable/SettingsTable';
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation';
import { useEditDialog } from '../../hooks/useEditDialog';
import { AxisDialog } from './components/AxisDialog/AxisDialog';
import styles from './VariantAxesSettings.module.scss';

const columns: Column<VariantAxis>[] = [
  { header: 'Eje', cell: (axis) => axis.name },
  { header: 'Valores', cell: (axis) => axis.values.join(', ') },
];

/** /configuracion/ejes: what the tenant's variants vary on (color, size...). Module variants. */
export function VariantAxesSettings() {
  const axes = useVariantAxes();
  const dialog = useEditDialog<VariantAxis>();
  const deletion = useDeleteConfirmation(useDeleteVariantAxis().mutateAsync);

  return (
    <SettingsSection
      title="Ejes de variante"
      description="En qué varían las variantes de un producto (color, tamaño, presentación…) y los valores que admite cada eje."
      action={
        <Button icon={<IconPlus />} onClick={dialog.openNew}>
          Agregar eje
        </Button>
      }
    >
      <QueryState
        queries={[axes]}
        errorTitle="No pudimos cargar los ejes"
        loading={<Spinner label="Cargando ejes…" />}
        isEmpty={axes.data?.length === 0}
        empty={<p className={styles.empty}>Todavía no hay ejes. Agregue uno para crear productos con variantes.</p>}
      >
        {() => (
          <SettingsTable
            caption="Ejes de variante"
            columns={columns}
            items={axes.data ?? []}
            nameOf={(axis) => axis.name}
            onEdit={dialog.openEdit}
            onDelete={deletion.ask}
          />
        )}
      </QueryState>
      <AxisDialog open={dialog.isOpen} axis={dialog.editing} onClose={dialog.close} />
      <ConfirmDeleteDialog
        {...deletion.dialogProps}
        title={`¿Borrar el eje ${deletion.target?.name ?? ''}?`}
        description="Un eje que alguna variante usa no se puede borrar."
      />
    </SettingsSection>
  );
}

import { useUnits, type UnitOfMeasure } from '@/features/catalog';
import { QueryState } from '@/shared/query';
import { Button, IconPlus, Spinner } from '@/shared/ui';
import { useDeleteUnit } from '../../api/units';
import { ConfirmDeleteDialog } from '../../components/ConfirmDeleteDialog/ConfirmDeleteDialog';
import { SettingsSection } from '../../components/SettingsSection/SettingsSection';
import { SettingsTable, type Column } from '../../components/SettingsTable/SettingsTable';
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation';
import { useEditDialog } from '../../hooks/useEditDialog';
import { UnitDialog } from './components/UnitDialog/UnitDialog';

const columns: Column<UnitOfMeasure>[] = [
  { header: 'Nombre', cell: (unit) => unit.name },
  { header: 'Código', cell: (unit) => unit.code, monospace: true },
];

/** /configuracion/unidades: the units the tenant counts and sells in. */
export function UnitsSettings() {
  const units = useUnits();
  const dialog = useEditDialog<UnitOfMeasure>();
  const deletion = useDeleteConfirmation(useDeleteUnit().mutateAsync);

  return (
    <SettingsSection
      title="Unidades de medida"
      description="En qué se cuentan y se venden sus productos. Una unidad que algún producto usa no se puede borrar."
      action={
        <Button icon={<IconPlus />} onClick={dialog.openNew}>
          Agregar unidad
        </Button>
      }
    >
      <QueryState
        queries={[units]}
        errorTitle="No pudimos cargar las unidades"
        loading={<Spinner label="Cargando unidades…" />}
      >
        {() => (
          <SettingsTable
            caption="Unidades de medida"
            columns={columns}
            items={units.data ?? []}
            nameOf={(unit) => unit.name}
            onEdit={dialog.openEdit}
            onDelete={deletion.ask}
          />
        )}
      </QueryState>
      <UnitDialog open={dialog.isOpen} unit={dialog.editing} onClose={dialog.close} />
      <ConfirmDeleteDialog
        {...deletion.dialogProps}
        title={`¿Borrar la unidad ${deletion.target?.name ?? ''}?`}
        description="Deja de estar disponible para los productos nuevos."
      />
    </SettingsSection>
  );
}

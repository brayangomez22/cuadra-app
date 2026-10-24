import { useAttributeDefinitions, type AttributeDefinition } from '@/features/catalog';
import { QueryState } from '@/shared/query';
import { Badge, Button, IconPlus, Spinner } from '@/shared/ui';
import { useDeleteAttribute } from '../../api/attributes';
import { ConfirmDeleteDialog } from '../../components/ConfirmDeleteDialog/ConfirmDeleteDialog';
import { SettingsSection } from '../../components/SettingsSection/SettingsSection';
import { SettingsTable, type Column } from '../../components/SettingsTable/SettingsTable';
import { useDeleteConfirmation } from '../../hooks/useDeleteConfirmation';
import { useEditDialog } from '../../hooks/useEditDialog';
import { describeType } from '../../model/attributeTypes';
import { AttributeDialog } from './components/AttributeDialog/AttributeDialog';
import styles from './AttributesSettings.module.scss';

const columns: Column<AttributeDefinition>[] = [
  { header: 'Nombre', cell: (attribute) => attribute.label },
  { header: 'Clave', cell: (attribute) => attribute.key, monospace: true },
  { header: 'Tipo', cell: describeType },
  {
    header: 'Obligatorio',
    cell: (attribute) =>
      attribute.required ? <Badge>Obligatorio</Badge> : <span className={styles.empty}>Opcional</span>,
  },
];

/** /configuracion/atributos: the fields the tenant's products have (brand, material...). */
export function AttributesSettings() {
  const attributes = useAttributeDefinitions();
  const dialog = useEditDialog<AttributeDefinition>();
  const deletion = useDeleteConfirmation(useDeleteAttribute().mutateAsync);

  return (
    <SettingsSection
      title="Atributos de producto"
      description="Los datos que registra de cada producto además del nombre y el precio: marca, material, principio activo…"
      action={
        <Button icon={<IconPlus />} onClick={dialog.openNew}>
          Agregar atributo
        </Button>
      }
    >
      <QueryState
        queries={[attributes]}
        errorTitle="No pudimos cargar los atributos"
        loading={<Spinner label="Cargando atributos…" />}
        isEmpty={attributes.data?.length === 0}
        empty={
          <p className={styles.empty}>Todavía no hay atributos. Sus productos solo llevan nombre, precio y unidad.</p>
        }
      >
        {() => (
          <SettingsTable
            caption="Atributos de producto"
            columns={columns}
            items={attributes.data ?? []}
            nameOf={(attribute) => attribute.label}
            onEdit={dialog.openEdit}
            onDelete={deletion.ask}
          />
        )}
      </QueryState>
      <AttributeDialog open={dialog.isOpen} attribute={dialog.editing} onClose={dialog.close} />
      <ConfirmDeleteDialog
        {...deletion.dialogProps}
        title={`¿Borrar el atributo ${deletion.target?.label ?? ''}?`}
        description="Un atributo que algún producto tiene no se puede borrar: primero quítele el valor a esos productos."
      />
    </SettingsSection>
  );
}

import { Button, EmptyState, IconBox, IconPlus } from '@/shared/ui';
import { CreateLocationDialog } from '../../../../components/CreateLocationDialog/CreateLocationDialog';

type Props = { canManage: boolean; onCreated: (locationId: string) => void };

/** Before the first location: the owner creates it; anyone else asks the owner. */
export function NoLocations({ canManage, onCreated }: Props) {
  return (
    <EmptyState
      icon={<IconBox />}
      title="Aún no hay sedes"
      description={
        canManage
          ? 'Cree la primera sede o bodega para registrar mercancía y ver sus existencias.'
          : 'Pida al dueño que cree la primera sede o bodega en Cuadra.'
      }
      action={
        canManage && (
          <CreateLocationDialog
            trigger={<Button icon={<IconPlus />}>Crear sede</Button>}
            onCreated={(created) => onCreated(created.id)}
          />
        )
      }
    />
  );
}

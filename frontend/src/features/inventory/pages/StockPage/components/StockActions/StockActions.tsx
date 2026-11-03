import { paths } from '@/app/routes/paths';
import { Button, ButtonLink, IconPlus } from '@/shared/ui';
import type { Location } from '../../../../api/types';
import { CreateLocationDialog } from '../../../../components/CreateLocationDialog/CreateLocationDialog';

type Props = {
  /** The location shown; without one there are no buttons about it. */
  location?: Location;
  canReceive: boolean;
  canCreate: boolean;
  onCreated: (locationId: string) => void;
};

/** The page's actions: what to reorder and register an entry, an adjustment or a transfer in the location shown, and create another location. */
export function StockActions({ location, canReceive, canCreate, onCreated }: Props) {
  return (
    <>
      {location && (
        <ButtonLink to={paths.inventory.reorder({ sede: location.id })} variant="secondary">
          Por reponer
        </ButtonLink>
      )}
      {location && canReceive && (
        <>
          <ButtonLink to={paths.inventory.adjustment({ sede: location.id })} variant="secondary">
            Registrar ajuste
          </ButtonLink>
          <ButtonLink to={paths.inventory.transfer({ sede: location.id })} variant="secondary">
            Trasladar
          </ButtonLink>
          <ButtonLink to={paths.inventory.receipt({ sede: location.id })} icon={<IconPlus />}>
            Registrar entrada
          </ButtonLink>
        </>
      )}
      {canCreate && (
        <CreateLocationDialog
          trigger={<Button variant="secondary">Nueva sede</Button>}
          onCreated={(created) => onCreated(created.id)}
        />
      )}
    </>
  );
}

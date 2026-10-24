import { paths } from '@/app/routes/paths';
import { Button, ButtonLink, IconPlus } from '@/shared/ui';
import type { Location } from '../../../../api/types';
import { CreateLocationDialog } from '../../../../components/CreateLocationDialog/CreateLocationDialog';

type Props = {
  /** Where an entry goes; without it (no location or no permission) there is no button. */
  location?: Location;
  canCreate: boolean;
  onCreated: (locationId: string) => void;
};

/** The page's actions: register an entry in the location shown and create another location. */
export function StockActions({ location, canCreate, onCreated }: Props) {
  return (
    <>
      {location && (
        <ButtonLink to={paths.inventory.receipt({ sede: location.id })} icon={<IconPlus />}>
          Registrar entrada
        </ButtonLink>
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

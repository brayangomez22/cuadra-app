import { paths } from '@/app/routes/paths';
import { ButtonLink, EmptyState, IconBox, IconPlus } from '@/shared/ui';
import type { Location } from '../../../../api/types';

type Props = { location: Location; canReceive: boolean };

/** A location without stock yet, with the way to register its first entry. */
export function EmptyStock({ location, canReceive }: Props) {
  return (
    <EmptyState
      icon={<IconBox />}
      title={`${location.name} aún no tiene existencias`}
      description="Cuando registre una entrada de mercancía, aquí verá cuánto hay de cada producto."
      action={
        canReceive && (
          <ButtonLink to={paths.inventory.receipt({ sede: location.id })} icon={<IconPlus />}>
            Registrar entrada
          </ButtonLink>
        )
      }
    />
  );
}

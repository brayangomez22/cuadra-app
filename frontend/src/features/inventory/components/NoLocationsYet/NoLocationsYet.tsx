import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { ButtonLink, EmptyState, IconBox, IconPlus } from '@/shared/ui';

type Props = {
  title: string;
  /** Why the screen needs a location, ending with a period. */
  reason: string;
};

/**
 * A screen with no location to work in: whoever creates locations (the
 * owner) goes to create the first one in the inventory; anyone else asks
 * the owner.
 */
export function NoLocationsYet({ title, reason }: Props) {
  const createsLocations = useCan()('tenant:manage');
  return (
    <EmptyState
      icon={<IconBox />}
      title={title}
      description={
        createsLocations ? `${reason} Créela en el inventario.` : `${reason} Pida al dueño que cree o active una.`
      }
      action={
        createsLocations && (
          <ButtonLink to={paths.inventory.stock()} icon={<IconPlus />}>
            Crear sede
          </ButtonLink>
        )
      }
    />
  );
}

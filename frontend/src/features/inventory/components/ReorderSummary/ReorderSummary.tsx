import { paths } from '@/app/routes/paths';
import { useCan } from '@/features/auth';
import { Alert, TextLink } from '@/shared/ui';
import { useReorderAlerts } from '../../api/reorder';

/**
 * How many products reached their minimum in every location, for the home
 * screen. Only the total is needed: one alert per page. Nothing while it
 * loads, when it fails or when there is nothing to reorder: it is a nudge,
 * not the list.
 */
export function ReorderSummary() {
  const can = useCan();
  const alerts = useReorderAlerts({ limit: 1, enabled: can('inventory:read') });
  const total = alerts.data?.total ?? 0;
  if (total === 0) return null;

  return (
    <Alert tone="warning" title={`${String(total)} ${total === 1 ? 'producto' : 'productos'} por reponer`}>
      Llegaron a su stock mínimo, sumando todas las sedes.{' '}
      <TextLink to={paths.inventory.reorder()}>Ver qué pedir</TextLink>
    </Alert>
  );
}

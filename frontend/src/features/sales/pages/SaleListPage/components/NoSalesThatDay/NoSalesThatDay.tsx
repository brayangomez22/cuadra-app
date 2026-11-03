import { paths } from '@/app/routes/paths';
import { formatDay, previousDay } from '@/shared/format';
import { ButtonLink, EmptyState, IconReceipt } from '@/shared/ui';

/** A day without sales, with the way to the day before: the demo and a slow morning both start empty. */
export function NoSalesThatDay({ day, location }: { day: string; location: string }) {
  const before = previousDay(day);
  return (
    <EmptyState
      icon={<IconReceipt />}
      title={`No hay ventas el ${formatDay(day)}`}
      description="Elija otro día arriba o vea el anterior."
      action={
        <ButtonLink to={paths.sales.list({ fecha: before, sede: location || undefined })} variant="secondary">
          {`Ver el ${formatDay(before)}`}
        </ButtonLink>
      }
    />
  );
}

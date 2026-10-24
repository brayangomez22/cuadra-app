import { paths } from '@/app/routes/paths';
import { ButtonLink, EmptyState, IconCash } from '@/shared/ui';

/** An open session has no report yet: it comes with the count, when it closes. */
export function SessionStillOpen() {
  return (
    <EmptyState
      icon={<IconCash />}
      title="La caja sigue abierta"
      description="El reporte aparece cuando quien la abrió la cierra con el conteo del efectivo."
      action={
        <ButtonLink to={paths.sales.cash()} variant="secondary">
          Ir a la caja
        </ButtonLink>
      }
    />
  );
}

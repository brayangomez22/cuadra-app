import { useCurrentUser } from '@/features/auth';
import { useUsers } from '@/features/users';
import { QueryState } from '@/shared/query';
import { EmptyState, IconHistory, Page, Pagination, Stack } from '@/shared/ui';
import { AUDIT_PAGE_SIZE, useAuditEvents } from '../../api/auditEvents';
import { AuditFilters } from './components/AuditFilters/AuditFilters';
import { AuditTable } from './components/AuditTable/AuditTable';
import { useAuditParams } from './hooks/useAuditParams';

/** /auditoria: who voided sales, changed prices, adjusted stock or credit, and when (only the owner). */
export function AuditPage() {
  const me = useCurrentUser();
  const params = useAuditParams();
  const events = useAuditEvents(params.query);
  const users = useUsers();
  const userNames = new Map((users.data ?? []).map((u) => [u.id, u.name]));
  if (me) userNames.set(me.id, me.name);

  return (
    <Page title="Auditoría" description="Quién anuló, descontó, cambió precios, ajustó inventario o cupos, y cuándo.">
      <Stack gap="4">
        <AuditFilters params={params} users={users.data ?? []} />
        <QueryState
          queries={[events]}
          errorTitle="No pudimos cargar el registro"
          isEmpty={events.data?.items.length === 0}
          empty={
            <EmptyState
              icon={<IconHistory />}
              title="Nada en el registro"
              description="Pruebe con otras fechas u otro filtro."
            />
          }
        >
          {() => (
            <>
              <AuditTable events={events.data?.items ?? []} userNames={userNames} />
              <Pagination
                label="Páginas del registro"
                total={events.data?.total ?? 0}
                limit={AUDIT_PAGE_SIZE}
                offset={params.offset}
                onOffsetChange={params.setOffset}
              />
            </>
          )}
        </QueryState>
      </Stack>
    </Page>
  );
}

import { enumParam, pageParam, stringParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { AUDIT_PAGE_SIZE, type AuditQuery } from '../../../api/auditEvents';
import { actionOptions } from '../../../model/actions';

const ALL = 'todas';
const actionParam = enumParam([ALL, ...actionOptions.map((option) => option.value)], ALL);
const textParam = stringParam();
// A new filter starts at the first page.
const resetPage = { reset: ['pagina'] };

/** The action, user, days and page of the audit log, kept in the URL (?accion=&usuario=&desde=&hasta=&pagina=). */
export function useAuditParams() {
  const [action, setAction] = useSearchParamsState('accion', actionParam, resetPage);
  const [user, setUser] = useSearchParamsState('usuario', textParam, resetPage);
  const [from, setFrom] = useSearchParamsState('desde', textParam, resetPage);
  const [to, setTo] = useSearchParamsState('hasta', textParam, resetPage);
  const [page, setPage] = useSearchParamsState('pagina', pageParam);

  const offset = (page - 1) * AUDIT_PAGE_SIZE;
  const query: AuditQuery = {
    action: action === ALL ? undefined : action,
    user_id: user || undefined,
    from: from || undefined,
    to: to || undefined,
    limit: AUDIT_PAGE_SIZE,
    offset,
  };
  return {
    query,
    action: action === ALL ? '' : action,
    setAction: (next: string) => setAction(actionParam.parse(next) ?? ALL),
    user,
    setUser,
    from,
    setFrom,
    to,
    setTo,
    offset,
    setOffset: (next: number) => setPage(next / AUDIT_PAGE_SIZE + 1),
  };
}

export type AuditParams = ReturnType<typeof useAuditParams>;

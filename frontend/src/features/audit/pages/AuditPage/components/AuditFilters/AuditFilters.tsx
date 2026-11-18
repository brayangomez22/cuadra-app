import type { User } from '@/features/users';
import { Field, Input, Select } from '@/shared/ui';
import { actionOptions } from '../../../../model/actions';
import type { AuditParams } from '../../hooks/useAuditParams';
import styles from './AuditFilters.module.scss';

// The value of "every action" and "every user" in the selects.
const ALL = 'todos';

type Props = { params: AuditParams; users: User[] };

/** The filters of the audit log: what was done, by whom and on which days (in Colombia). */
export function AuditFilters({ params, users }: Props) {
  const userOptions = [{ value: ALL, label: 'Todos' }, ...users.map((u) => ({ value: u.id, label: u.name }))];
  return (
    <div role="search" aria-label="Filtrar el registro" className={styles.root}>
      <Field label="Acción">
        <Select
          options={[{ value: ALL, label: 'Todas' }, ...actionOptions]}
          value={params.action || ALL}
          onValueChange={(value) => params.setAction(value === ALL ? '' : value)}
        />
      </Field>
      <Field label="Usuario">
        <Select
          options={userOptions}
          value={params.user || ALL}
          onValueChange={(value) => params.setUser(value === ALL ? '' : value)}
        />
      </Field>
      <Field label="Desde">
        <Input type="date" value={params.from} onChange={(event) => params.setFrom(event.target.value)} />
      </Field>
      <Field label="Hasta">
        <Input type="date" value={params.to} onChange={(event) => params.setTo(event.target.value)} />
      </Field>
    </div>
  );
}

import type { ReactNode } from 'react';
import { useLocations } from '@/features/inventory';
import { Alert, Field, Input, Select, Stack } from '@/shared/ui';
import type { ReportParams } from '../../hooks/useReportParams';
import styles from './ReportFilters.module.scss';

// The value of "every location" in the select.
const ALL = 'todas';

type Props = {
  params: ReportParams;
  /** Without a period: the idle stock looks back from today. */
  period?: boolean;
  /** The report's own filters (grouping, order, category...). */
  children?: ReactNode;
};

/** The days (in Colombia) and the location of a report, and its own filters. */
export function ReportFilters({ params, period = true, children }: Props) {
  const locations = useLocations();
  const options = [
    { value: ALL, label: 'Todas' },
    ...(locations.data ?? []).map((l) => ({ value: l.id, label: l.name })),
  ];
  return (
    <Stack gap="3">
      <div role="search" aria-label="Filtrar el reporte" className={styles.root}>
        {period && (
          <>
            <Field label="Desde">
              <Input type="date" value={params.from} onChange={(event) => params.setFrom(event.target.value)} />
            </Field>
            <Field label="Hasta">
              <Input type="date" value={params.to} onChange={(event) => params.setTo(event.target.value)} />
            </Field>
          </>
        )}
        <Field label="Sede">
          <Select
            options={options}
            value={params.location || ALL}
            onValueChange={(value) => params.setLocation(value === ALL ? '' : value)}
          />
        </Field>
        {children}
      </div>
      {period && params.problem && <Alert tone="warning">{params.problem}</Alert>}
    </Stack>
  );
}

import { Navigate, Outlet } from 'react-router';
import { paths } from '@/app/routes/paths';
import { useAccess, visibleItems } from '@/features/auth';
import { SectionLayout } from '@/shared/ui';
import { useReportParams } from '../../hooks/useReportParams';
import { reportSections } from '../../model/sections';

/** /reportes: the business reports the user may see, and the chosen one, on the same days and location. */
export function ReportsLayout() {
  const { links } = useReportParams();
  return (
    <SectionLayout
      title="Reportes"
      description="Cuánto vendió, qué deja plata y qué no, y lo que hizo cada persona del equipo."
      label="Reportes"
      sections={visibleItems(reportSections(links), useAccess())}
    >
      <Outlet />
    </SectionLayout>
  );
}

/** /reportes opens the sales report, keeping the period and location of the link. */
export function ReportsHome() {
  const { links } = useReportParams();
  return <Navigate to={paths.reports.sales(links)} replace />;
}

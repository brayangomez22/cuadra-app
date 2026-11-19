import { paths } from '@/app/routes/paths';
import type { Gated } from '@/features/auth';
import { COST_PERMISSION } from './costs';

/** The query a report link keeps: the period and the location. */
export type ReportLinks = { desde?: string; hasta?: string; sede?: string };

/** The reports, in the order of the menu; profit by sale only for whoever sees costs. */
export function reportSections(links: ReportLinks): (Gated & { to: string; label: string })[] {
  return [
    { to: paths.reports.sales(links), label: 'Ventas' },
    { to: paths.reports.products(links), label: 'Productos' },
    { to: paths.reports.profits(links), label: 'Utilidad por venta', permission: COST_PERMISSION },
    { to: paths.reports.idle({ sede: links.sede }), label: 'Sin movimiento' },
    { to: paths.reports.cash(links), label: 'Cierres de caja' },
    { to: paths.reports.team(links), label: 'Control del equipo' },
    { to: paths.reports.adjustments(links), label: 'Ajustes de inventario' },
  ];
}

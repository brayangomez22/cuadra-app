import { useCan } from '@/features/auth';
import { QueryState } from '@/shared/query';
import { EmptyState, IconTag, Pagination } from '@/shared/ui';
import { useProductsReport } from '../../api/reports';
import { ExportButtons } from '../../components/ExportButtons/ExportButtons';
import { ReportFilters } from '../../components/ReportFilters/ReportFilters';
import { ReportSection } from '../../components/ReportSection/ReportSection';
import { useReportParams } from '../../hooks/useReportParams';
import { COST_PERMISSION } from '../../model/costs';
import { ProductFilters } from './components/ProductFilters/ProductFilters';
import { ProductsTable } from './components/ProductsTable/ProductsTable';
import { PRODUCTS_PAGE_SIZE, useProductsParams } from './hooks/useProductsParams';

/** /reportes/productos: what each product sold, by variant, and what it left (for whoever sees costs). */
export function ProductsReportPage() {
  const params = useReportParams();
  const products = useProductsParams();
  const seesCosts = useCan()(COST_PERMISSION);
  const filters = { ...params.query, ...products.query };
  const report = useProductsReport({ ...filters, limit: PRODUCTS_PAGE_SIZE, offset: products.offset }, !params.problem);

  return (
    <ReportSection
      title="Productos vendidos"
      description={
        seesCosts
          ? 'Lo vendido por producto y variante, con la utilidad al costo del momento de cada venta.'
          : 'Lo vendido por producto y variante.'
      }
      actions={<ExportButtons query={{ report: 'products', ...filters }} disabled={Boolean(params.problem)} />}
    >
      <ReportFilters params={params}>
        <ProductFilters params={products} seesCosts={seesCosts} />
      </ReportFilters>
      {!params.problem && (
        <QueryState
          queries={[report]}
          errorTitle="No pudimos cargar los productos vendidos"
          isEmpty={report.data?.items.length === 0}
          empty={
            <EmptyState icon={<IconTag />} title="Nada vendido" description="Pruebe con otras fechas o filtros." />
          }
        >
          {() => (
            <>
              <ProductsTable rows={report.data?.items ?? []} seesCosts={seesCosts} />
              <Pagination
                label="Páginas de productos"
                total={report.data?.total ?? 0}
                limit={PRODUCTS_PAGE_SIZE}
                offset={products.offset}
                onOffsetChange={products.setOffset}
              />
            </>
          )}
        </QueryState>
      )}
    </ReportSection>
  );
}

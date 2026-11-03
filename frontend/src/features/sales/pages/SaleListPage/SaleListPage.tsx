import { useLocations } from '@/features/inventory';
import { QueryState } from '@/shared/query';
import { Page, Pagination, Spinner } from '@/shared/ui';
import { SALE_PAGE_SIZE, useSales } from '../../api/sales';
import { NoSalesThatDay } from './components/NoSalesThatDay/NoSalesThatDay';
import { SaleDaySummary } from './components/SaleDaySummary/SaleDaySummary';
import { SaleFilters } from './components/SaleFilters/SaleFilters';
import { SaleTable } from './components/SaleTable/SaleTable';
import { useSaleListParams } from './hooks/useSaleListParams';
import styles from './SaleListPage.module.scss';

/** The sales at /ventas: those of a day, voided ones too, to open, reprint or void them. */
export function SaleListPage() {
  const params = useSaleListParams();
  const sales = useSales(params.query);
  const locations = useLocations().data ?? [];
  const locationName = (id: string) => locations.find((l) => l.id === id)?.name ?? '';
  return (
    <Page title="Ventas" description="Lo que se vendió cada día: abra una venta para reimprimir la tirilla o anularla.">
      <SaleFilters params={params} locations={locations} />
      <QueryState
        queries={[sales]}
        errorTitle="No pudimos cargar las ventas"
        loading={<Spinner label="Cargando ventas…" className={styles.loading} />}
        isEmpty={sales.data?.items.length === 0}
        empty={<NoSalesThatDay day={params.day} location={params.location} />}
      >
        {() =>
          sales.data && (
            <div className={styles.results} aria-busy={sales.isPlaceholderData || undefined}>
              <SaleDaySummary page={sales.data} />
              <SaleTable sales={sales.data.items} locationName={locationName} />
              <Pagination
                label="Páginas de ventas"
                total={sales.data.total}
                limit={SALE_PAGE_SIZE}
                offset={params.offset}
                onOffsetChange={params.setOffset}
              />
            </div>
          )
        }
      </QueryState>
    </Page>
  );
}

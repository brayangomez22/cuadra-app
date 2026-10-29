import { pageParam, useSearchParamsState } from '@/shared/hooks/useSearchParamsState';
import { QueryState } from '@/shared/query';
import { Pagination, Table, TableBody, TableHead, TableHeaderCell, TableRow } from '@/shared/ui';
import { PAGE_SIZE, useCreditMovements } from '../../../../api/movements';
import { StatementRow } from './StatementRow';
import styles from './Statement.module.scss';

type Props = { customerId: string };

/** The customer's statement: credit sales, payments and voided credit sales, each with the balance it left. */
export function Statement({ customerId }: Props) {
  const [page, setPage] = useSearchParamsState('pagina', pageParam);
  const offset = (page - 1) * PAGE_SIZE;
  const movements = useCreditMovements(customerId, { limit: PAGE_SIZE, offset });
  return (
    <QueryState
      queries={[movements]}
      errorTitle="No pudimos cargar el estado de cuenta"
      isEmpty={movements.data?.items.length === 0}
      empty={<p className={styles.empty}>Todavía no tiene ventas a crédito ni abonos.</p>}
    >
      {() =>
        movements.data && (
          <div className={styles.root} aria-busy={movements.isPlaceholderData || undefined}>
            <Table caption="Estado de cuenta" showCaption>
              <TableHead>
                <TableRow>
                  <TableHeaderCell>Fecha</TableHeaderCell>
                  <TableHeaderCell>Movimiento</TableHeaderCell>
                  <TableHeaderCell numeric>Valor</TableHeaderCell>
                  <TableHeaderCell numeric>Saldo</TableHeaderCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {movements.data.items.map((movement) => (
                  <StatementRow key={`${movement.kind}-${movement.at}`} movement={movement} />
                ))}
              </TableBody>
            </Table>
            <Pagination
              label="Páginas del estado de cuenta"
              total={movements.data.total}
              limit={PAGE_SIZE}
              offset={offset}
              onOffsetChange={(next) => setPage(next / PAGE_SIZE + 1)}
            />
          </div>
        )
      }
    </QueryState>
  );
}

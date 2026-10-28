import { useParams } from 'react-router';
import { QueryState } from '@/shared/query';
import { Alert, Page, Spinner } from '@/shared/ui';
import { useLocations } from '../../api/locations';
import { useStockCount } from '../../api/stockCounts';
import { countedText } from '../../model/stockCount/countSummary';
import { CountedEntries } from './components/CountedEntries/CountedEntries';
import { CountEntryForm } from './components/CountEntryForm/CountEntryForm';
import styles from './CountingPage.module.scss';

const closedMessages = {
  approved: 'Este conteo ya se aprobó: no se puede contar más.',
  canceled: 'Este conteo se canceló: no se puede contar más.',
};

/**
 * /inventario/conteos/:countId: where people count, from a phone or a
 * counter. It never shows the expected stock: the count is blind.
 */
export function CountingPage() {
  const { countId = '' } = useParams();
  const count = useStockCount(countId);
  const locations = useLocations();
  const location = locations.data?.find((l) => l.id === count.data?.location_id);

  return (
    <Page
      title={location ? `Conteo de ${location.name}` : 'Conteo'}
      description={count.data && countedText(count.data)}
    >
      <QueryState
        queries={[count, locations]}
        errorTitle="No pudimos cargar el conteo"
        loading={<Spinner label="Cargando el conteo…" className={styles.loading} />}
      >
        {() =>
          count.data && (
            <div className={styles.root}>
              {count.data.status === 'open' ? (
                <CountEntryForm countId={countId} />
              ) : (
                <Alert tone="info">{closedMessages[count.data.status]}</Alert>
              )}
              <CountedEntries countId={countId} open={count.data.status === 'open'} />
            </div>
          )
        }
      </QueryState>
    </Page>
  );
}

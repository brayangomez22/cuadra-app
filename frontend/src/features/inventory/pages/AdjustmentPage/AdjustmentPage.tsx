import { Page } from '@/shared/ui';
import { StockFormState } from '../../components/StockFormState/StockFormState';
import { useStockFormSetup } from '../../hooks/useStockFormSetup';
import { AdjustmentForm } from './components/AdjustmentForm/AdjustmentForm';

/** /inventario/ajuste: corrects a location's stock (something broke, got lost, turned up...). */
export function AdjustmentPage() {
  const setup = useStockFormSetup();
  return (
    <Page
      title="Registrar ajuste"
      description="Corrige la existencia de una sede: algo se rompió, se perdió, apareció o el conteo no da. Queda en el kardex con su motivo y no cambia el costo."
    >
      <StockFormState
        setup={setup}
        emptyTitle="Aún no hay sedes activas"
        emptyDescription="Para ajustar la existencia primero debe existir una sede o bodega."
      >
        {() => (
          <AdjustmentForm
            locations={setup.options}
            initialItem={setup.initialItem}
            initialLocation={setup.initialLocation}
          />
        )}
      </StockFormState>
    </Page>
  );
}

import { Page } from '@/shared/ui';
import { StockFormState } from '../../components/StockFormState/StockFormState';
import { useStockFormSetup } from '../../hooks/useStockFormSetup';
import { TransferForm } from './components/TransferForm/TransferForm';

/** /inventario/traslado: goods that move between two locations at once, with no shipment in between. */
export function TransferPage() {
  const setup = useStockFormSetup();
  return (
    <Page
      title="Trasladar mercancía"
      description="Pasa mercancía de una sede a otra en un solo paso. Llega con el mismo costo que tenía en la sede de donde sale."
    >
      <StockFormState
        setup={setup}
        minLocations={2}
        emptyTitle="Hacen falta dos sedes activas"
        emptyDescription="Para trasladar mercancía la empresa debe tener al menos dos sedes o bodegas activas."
      >
        {() => (
          <TransferForm
            locations={setup.options}
            initialItem={setup.initialItem}
            initialLocation={setup.initialLocation}
          />
        )}
      </StockFormState>
    </Page>
  );
}

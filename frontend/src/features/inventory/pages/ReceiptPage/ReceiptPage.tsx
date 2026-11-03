import { Page } from '@/shared/ui';
import { StockFormState } from '../../components/StockFormState/StockFormState';
import { useStockFormSetup } from '../../hooks/useStockFormSetup';
import { ReceiptForm } from './components/ReceiptForm/ReceiptForm';

/** /inventario/entrada: goods that arrive at a location. */
export function ReceiptPage() {
  const setup = useStockFormSetup();
  return (
    <Page
      title="Registrar entrada"
      description="La mercancía que llega suma a la existencia de la sede y actualiza el costo promedio."
    >
      <StockFormState
        setup={setup}
        emptyTitle="Aún no hay sedes activas"
        emptyDescription="Para registrar mercancía primero debe existir una sede o bodega."
      >
        {() => (
          <ReceiptForm
            locations={setup.options}
            initialItem={setup.initialItem}
            initialLocation={setup.initialLocation}
          />
        )}
      </StockFormState>
    </Page>
  );
}

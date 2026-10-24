import { aMixedSale, renderReceipt } from '../../testing/receipt';

describe('Receipt', () => {
  it('aclara que no es una factura electrónica', () => {
    const receipt = renderReceipt();

    expect(receipt.getByText('Este documento no es una factura electrónica de venta.')).toBeInTheDocument();
  });

  it('marca como ANULADA una venta anulada', () => {
    const receipt = renderReceipt(
      aMixedSale({
        status: 'voided',
        void: { user_id: '01920000-0000-7000-8000-000000000001', reason: 'Error', voided_at: '2026-10-11T16:00:00Z' },
      }),
    );

    expect(receipt.getByText('ANULADA')).toBeInTheDocument();
  });

  it('una venta confirmada no dice ANULADA', () => {
    const receipt = renderReceipt();

    expect(receipt.queryByText('ANULADA')).not.toBeInTheDocument();
  });
});

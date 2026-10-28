import { figure, renderReceipt } from '../../testing/receipt';

describe('Receipt', () => {
  it.each([
    // Ferretería: un maestro de obra con cédula; almacén de ropa: una empresa mayorista.
    { name: 'Carlos Pérez', document: 'CC 1020304050' },
    { name: 'Confecciones Sol SAS', document: 'NIT 901234567-7' },
  ])('muestra el cliente $name con su documento', (customer) => {
    const receipt = renderReceipt(undefined, customer);

    expect(figure(receipt, 'Cliente')).toBe(customer.name);
    expect(figure(receipt, 'Documento')).toBe(customer.document);
  });

  it('sin cliente dice consumidor final', () => {
    const receipt = renderReceipt();

    expect(figure(receipt, 'Cliente')).toBe('Consumidor final');
    expect(receipt.queryByText('Documento')).not.toBeInTheDocument();
  });
});

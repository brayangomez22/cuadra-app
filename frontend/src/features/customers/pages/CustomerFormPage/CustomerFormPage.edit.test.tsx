import { screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { json } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { customersApi } from '../../testing/customersApi';
import { aBoutique, customerIds, customerInput } from '@/test/fixtures';

const url = `/api/v1/customers/${customerIds.boutique}`;

describe('CustomerFormPage: editar', () => {
  it('carga el cliente, lo desactiva y lo guarda', async () => {
    const api = customersApi({
      routes: {
        [`GET ${url}`]: json(200, aBoutique()),
        [`PUT ${url}`]: json(200, aBoutique({ active: false })),
      },
    });
    renderApp(`/clientes/${customerIds.boutique}/editar`);

    expect(await screen.findByLabelText('Razón social')).toHaveValue('Confecciones Sol SAS');
    expect(screen.getByLabelText('Nombre comercial (opcional)')).toHaveValue('Boutique Sol');
    expect(screen.getByRole('checkbox', { name: /Gran contribuyente/ })).toBeChecked();
    await userEvent.click(screen.getByRole('checkbox', { name: 'Activo' }));
    await userEvent.click(screen.getByRole('button', { name: 'Guardar cliente' }));

    await waitFor(() => expect(api.calls(`PUT ${url}`)).toHaveLength(1));
    expect(await api.calls(`PUT ${url}`)[0]!.json()).toEqual(customerInput(aBoutique({ active: false })));
  });
});

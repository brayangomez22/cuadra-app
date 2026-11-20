import { screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, fakeApi, json, signedIn } from '@/test/fakeApi';
import { renderApp } from '@/test/renderApp';
import { aChecklist, clothingSteps, hardwareSteps } from '../../testing/checklists';

const checklistKey = 'GET /api/v1/onboarding/checklist';

describe('GettingStarted', () => {
  it('muestra el progreso y los pasos pendientes con su acción', async () => {
    fakeApi({
      ...signedIn(aUser('owner')),
      [checklistKey]: json(200, aChecklist(hardwareSteps, ['business_profile', 'products'])),
    });
    renderApp('/');

    const card = await screen.findByRole('region', { name: 'Primeros pasos' });
    expect(within(card).getByText('2 de 7 listos')).toBeInTheDocument();
    const steps = within(card).getAllByRole('listitem');
    expect(steps).toHaveLength(7);
    expect(steps[1]).toHaveTextContent('Listo');
    expect(within(steps[1]!).queryByRole('link')).not.toBeInTheDocument();
    expect(within(steps[2]!).getByRole('link', { name: 'Ver productos' })).toHaveAttribute('href', '/productos');
    expect(within(steps[6]!).getByRole('link', { name: 'Ir a la caja' })).toHaveAttribute('href', '/caja');
  });

  it('dos tipos de negocio muestran sus pasos propios', async () => {
    fakeApi({ ...signedIn(aUser('owner')), [checklistKey]: json(200, aChecklist(clothingSteps)) });
    renderApp('/');

    const card = await screen.findByRole('region', { name: 'Primeros pasos' });
    expect(within(card).getByText('Cree un producto con variantes')).toBeInTheDocument();
    expect(within(card).queryByText('Venda en otras unidades')).not.toBeInTheDocument();
    expect(within(card).getByRole('link', { name: 'Importar desde Excel' })).toHaveAttribute(
      'href',
      '/productos/importar',
    );
  });

  it('al ocultar desaparece', async () => {
    const api = fakeApi({
      ...signedIn(aUser('owner')),
      [checklistKey]: json(200, aChecklist(hardwareSteps)),
      'PUT /api/v1/onboarding/checklist/dismissal': new Response(null, { status: 204 }),
    });
    renderApp('/');

    await userEvent.click(await screen.findByRole('button', { name: 'Ocultar primeros pasos' }));

    await waitFor(() => expect(screen.queryByRole('region', { name: 'Primeros pasos' })).not.toBeInTheDocument());
    expect(api.calls('PUT /api/v1/onboarding/checklist/dismissal')).toHaveLength(1);
  });

  it('no aparece si la API dice que no es visible', async () => {
    const api = fakeApi({
      ...signedIn(aUser('cashier')),
      [checklistKey]: json(200, aChecklist(['first_sale'], ['first_sale'])),
    });
    renderApp('/');

    expect(await screen.findByRole('heading', { name: /Hola/ })).toBeInTheDocument();
    await waitFor(() => expect(api.calls(checklistKey)).toHaveLength(1));
    expect(screen.queryByRole('region', { name: 'Primeros pasos' })).not.toBeInTheDocument();
  });

  it('el paso de la facturación vuelve marcado como importante', async () => {
    fakeApi({
      ...signedIn(aUser('owner')),
      [checklistKey]: json(
        200,
        aChecklist(['invoicing', 'first_sale'], ['first_sale'], { dismissed: true, visible: true }),
      ),
    });
    renderApp('/');

    const card = await screen.findByRole('region', { name: 'Primeros pasos' });
    expect(within(card).getByText('Importante')).toBeInTheDocument();
    expect(within(card).getByText('Registre la resolución de facturación')).toBeInTheDocument();
  });
});

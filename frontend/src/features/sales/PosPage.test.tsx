import { cleanup, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { aUser, apiError, fakeApi, json, signedIn, type Me } from '@/test/fakeApi';
import { aCement, aLocation, aProduct, aSale, ids, page, plain, query, type Product } from '@/test/fixtures';
import { renderApp } from '@/test/renderApp';

const CEMENT_BARCODE = '7701234567890';

type Options = {
  user?: Me;
  limit?: { unlimited: boolean; max: string };
  locations?: ReturnType<typeof aLocation>[];
  extra?: Parameters<typeof fakeApi>[0];
};

// The products answer like the API: an exact SKU or barcode, or every word in the name.
function search(products: Product[], q: string) {
  const term = q.toLowerCase();
  return products.filter(
    (p) => p.barcode === q || p.sku.toLowerCase() === term || p.name.toLowerCase().includes(term),
  );
}

function posApi({
  user = aUser('cashier'),
  limit = { unlimited: false, max: '0' },
  locations = [aLocation()],
  extra = {},
}: Options = {}) {
  const products = [aProduct(), aCement({ barcode: CEMENT_BARCODE })];
  return fakeApi({
    ...signedIn(user),
    'GET /api/v1/locations': json(200, { items: locations }),
    'GET /api/v1/sales/discount-limit': json(200, limit),
    'GET /api/v1/products': (request) => json(200, page(search(products, query(request).get('q') ?? ''))),
    'POST /api/v1/sales': json(201, aSale()),
    ...extra,
  });
}

const user = () => userEvent.setup({ delay: null });

async function searchBox() {
  return screen.findByRole('combobox', { name: 'Buscar producto' });
}

// scan types a code and Enter into the search, as a barcode reader does.
async function scan(code: string) {
  await user().type(await searchBox(), `${code}{Enter}`);
}

function cartLine(name: RegExp | string) {
  return within(screen.getByRole('listitem', { name }));
}

async function findCartLine(name: RegExp | string) {
  return within(await screen.findByRole('listitem', { name }));
}

// summaryValue reads an amount of the sale's summary by its term ("Total", "IVA").
function summaryValue(term: string) {
  const summary = within(screen.getByRole('region', { name: 'Resumen de la venta' }));
  return plain(summary.getByText(term, { selector: 'dt' }).nextElementSibling?.textContent);
}

// posted reads the body of the sale sent to the API.
async function posted(api: ReturnType<typeof posApi>) {
  const [request] = api.calls('POST /api/v1/sales');
  expect(request).toBeDefined();
  return request!.json();
}

const checkout = () => screen.findByRole('dialog', { name: 'Cobrar' });

beforeEach(() => {
  sessionStorage.clear();
  localStorage.clear();
});

describe('PosPage', () => {
  it('escanear el mismo código dos veces suma cantidad', async () => {
    posApi();
    renderApp('/vender');

    await scan(CEMENT_BARCODE);
    await findCartLine(/Cemento gris 50kg/);
    await scan(CEMENT_BARCODE);

    await waitFor(() => expect(cartLine(/Cemento gris 50kg/).getByLabelText(/^Cantidad/)).toHaveValue('2'));
    expect(screen.getAllByRole('listitem', { name: /Cemento/ })).toHaveLength(1);
    expect(summaryValue('Total')).toBe('$ 76.160');
    // The reader can scan the next product at once.
    expect(await searchBox()).toHaveFocus();
    expect(await searchBox()).toHaveValue('');
  });

  it('agrega con la cantidad antes del código (3*CEM-50)', async () => {
    posApi();
    renderApp('/vender');

    await scan('3*CEM-50');

    expect((await findCartLine(/Cemento/)).getByLabelText(/^Cantidad/)).toHaveValue('3');
  });

  it('busca por nombre y elige entre varios resultados con las flechas', async () => {
    posApi();
    renderApp('/vender');

    await user().type(await searchBox(), 'o');
    expect(await screen.findAllByRole('option')).toHaveLength(2);
    await user().keyboard('{ArrowDown}{ArrowDown}{Enter}');

    await findCartLine(/Cemento gris 50kg/);
    expect(screen.queryByRole('listitem', { name: /Tubo PVC/ })).not.toBeInTheDocument();
  });

  it('avisa cuando no encuentra el código', async () => {
    posApi();
    renderApp('/vender');

    await scan('0000000');

    expect(await screen.findByText(/No encontramos «0000000»/)).toBeInTheDocument();
  });

  it('edita una cantidad decimal', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');

    const quantity = (await findCartLine(/Tubo PVC/)).getByLabelText(/^Cantidad/);
    await user().clear(quantity);
    await user().type(quantity, '2,5');

    // 2.5 × 10.000 = 25.000 + IVA 19 %.
    await waitFor(() => expect(summaryValue('Total')).toBe('$ 29.750'));
    expect(summaryValue('IVA')).toBe('$ 4.750');
  });

  it('no deja cobrar con una cantidad inválida', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');

    await user().clear((await findCartLine(/Tubo PVC/)).getByLabelText(/^Cantidad/));

    expect(cartLine(/Tubo PVC/).getByLabelText(/^Cantidad/)).toHaveAccessibleDescription(/Escriba la cantidad/);
    expect(screen.getByRole('button', { name: 'Cobrar' })).toBeDisabled();
  });

  it('dice qué corregir cuando una línea no deja cobrar', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');
    await scan(CEMENT_BARCODE);

    await user().clear((await findCartLine(/Tubo PVC/)).getByLabelText(/^Cantidad/));

    expect(screen.getByRole('button', { name: 'Cobrar' })).toBeDisabled();
    expect(screen.getByText('Corrija 1 línea para cobrar.')).toBeInTheDocument();
    expect(cartLine(/Tubo PVC/).getByText('Sin total')).toBeInTheDocument();
  });

  it('quita una línea del carrito', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');

    await user().click((await findCartLine(/Tubo PVC/)).getByRole('button', { name: /Quitar/ }));

    expect(screen.queryByRole('listitem', { name: /Tubo PVC/ })).not.toBeInTheDocument();
    expect(await searchBox()).toHaveFocus();
  });

  describe('cobro', () => {
    it('no permite cobrar con pago insuficiente', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      const dialog = within(await checkout());
      await user().type(dialog.getByLabelText('Valor del pago 1'), '10000{Enter}');

      expect(plain(dialog.getByRole('alert').textContent)).toContain('Faltan $ 1.900');
      expect(api.calls('POST /api/v1/sales')).toHaveLength(0);
    });

    it('confirma con Enter, envía el carrito y muestra el cambio', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan(CEMENT_BARCODE);
      await findCartLine(/Cemento/);
      await scan(CEMENT_BARCODE);
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      const dialog = within(await checkout());
      expect(dialog.getByLabelText('Valor del pago 1')).toHaveFocus();
      await user().type(dialog.getByLabelText('Valor del pago 1'), '100000');
      expect(plain(dialog.getByText('Cambio', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 11.940');
      await user().keyboard('{Enter}');

      expect(await screen.findByRole('dialog', { name: 'Venta N.º 15' })).toBeInTheDocument();
      // The change of the API's answer, not the preview's.
      expect(plain(screen.getByRole('dialog').textContent)).toContain('$ 8.100');
      expect(await posted(api)).toEqual({
        location_id: ids.main,
        lines: [
          { product_id: ids.cement, quantity: '2' },
          { product_id: ids.pipe, quantity: '1' },
        ],
        payments: [{ method: 'cash', amount: '100000' }],
      });

      expect(screen.getByRole('button', { name: 'Nueva venta' })).toHaveFocus();
      await user().keyboard('{Enter}');
      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(screen.queryByRole('listitem', { name: /Cemento/ })).not.toBeInTheDocument();
      expect(await searchBox()).toHaveFocus();
    });

    it('acepta el valor con puntos de miles, como lo escribe el cajero', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      const dialog = within(await checkout());
      await user().type(dialog.getByLabelText('Valor del pago 1'), '20.000');
      expect(plain(dialog.getByText('Cambio', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 8.100');
      await user().keyboard('{Enter}');

      await screen.findByRole('dialog', { name: 'Venta N.º 15' });
      expect((await posted(api)).payments).toEqual([{ method: 'cash', amount: '20000' }]);
    });

    it('muestra cuánto falta mientras se escribe el pago', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      const dialog = within(await checkout());
      await user().type(dialog.getByLabelText('Valor del pago 1'), '10000');

      expect(plain(dialog.getByText('Falta', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 1.900');
      expect(dialog.queryByText('Cambio', { selector: 'dt' })).not.toBeInTheDocument();
    });

    it('sin escribir el valor, cobra el total exacto en efectivo', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');

      await screen.findByRole('dialog', { name: 'Venta N.º 15' });
      expect((await posted(api)).payments).toEqual([
        { method: 'cash', amount: '11900.00' },
      ]);
    });

    it('pago mixto calcula el cambio solo con el efectivo', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      const dialog = within(await checkout());

      await user().type(dialog.getByLabelText('Valor del pago 1'), '10000');
      await user().click(dialog.getByRole('button', { name: 'Agregar medio de pago' }));
      // The new payment proposes what is missing.
      expect(dialog.getByLabelText('Valor del pago 2')).toHaveValue('1900');
      expect(dialog.getByRole('combobox', { name: 'Medio de pago 2' })).toHaveTextContent('Tarjeta');

      const cash = dialog.getByLabelText('Valor del pago 1');
      await user().clear(cash);
      await user().type(cash, '20000');
      expect(plain(dialog.getByText('Cambio', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 10.000');
      await user().click(dialog.getByRole('button', { name: 'Confirmar venta' }));

      await screen.findByRole('dialog', { name: 'Venta N.º 15' });
      expect((await posted(api)).payments).toEqual([
        { method: 'cash', amount: '20000' },
        { method: 'card', amount: '1900' },
      ]);
    });

    it('rechaza un sobrepago con medios distintos del efectivo', async () => {
      const api = posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      const dialog = within(await checkout());

      await user().click(dialog.getByRole('combobox', { name: 'Medio de pago 1' }));
      await user().click(await screen.findByRole('option', { name: 'Nequi' }));
      await user().type(dialog.getByLabelText('Valor del pago 1'), '15000');
      await user().click(dialog.getByRole('button', { name: 'Confirmar venta' }));

      expect(dialog.getByRole('alert')).toHaveTextContent(/Solo el efectivo da cambio/);
      expect(api.calls('POST /api/v1/sales')).toHaveLength(0);
    });

    it('marca la línea sin stock cuando la API responde insufficient_stock', async () => {
      posApi({
        extra: {
          'POST /api/v1/sales': apiError(409, 'insufficient_stock', 'No hay stock suficiente de Cemento gris 50kg.', {
            product_id: ids.cement,
          }),
        },
      });
      renderApp('/vender');
      await scan(CEMENT_BARCODE);
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');

      expect(await within(await checkout()).findByRole('alert')).toHaveTextContent(
        'No hay stock suficiente de Cemento gris 50kg.',
      );
      // Back to the cart, the line to fix is marked.
      await user().keyboard('{Escape}');
      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(cartLine(/Cemento/).getByText('Sin stock suficiente')).toBeInTheDocument();
      expect(cartLine(/Tubo PVC/).queryByText('Sin stock suficiente')).not.toBeInTheDocument();
    });
  });

  describe('tirilla', () => {
    // sellPipe charges a pipe in cash and waits for the sale's result.
    async function sellPipe() {
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');
      return within(await screen.findByRole('dialog', { name: 'Venta N.º 15' }));
    }

    // The printable receipt lives outside the app and is hidden on screen.
    const printed = () => screen.getByRole('article', { name: 'Tirilla de la venta N.º 15', hidden: true });

    it('después de confirmar imprime la tirilla con los datos del negocio y de la venta', async () => {
      const print = vi.spyOn(window, 'print').mockImplementation(() => undefined);
      posApi();
      renderApp('/vender');

      const done = await sellPipe();
      await user().click(done.getByRole('button', { name: 'Imprimir tirilla' }));

      expect(print).toHaveBeenCalledOnce();
      const receipt = within(printed());
      expect(receipt.getByText('NIT 890903938-8')).toBeInTheDocument();
      expect(receipt.getByText('Bodega principal')).toBeInTheDocument();
      expect(plain(receipt.getByText('Total', { selector: 'dt' }).nextElementSibling?.textContent)).toBe('$ 11.900');
      // The next sale is still one Enter away.
      expect(done.getByRole('button', { name: 'Nueva venta' })).toBeInTheDocument();
    });

    it('sin venta confirmada no hay tirilla para imprimir', async () => {
      posApi();
      renderApp('/vender');
      await searchBox();

      expect(screen.queryByRole('article', { hidden: true })).not.toBeInTheDocument();
    });

    it('usa papel de 80 mm por defecto y recuerda el ancho elegido en el equipo', async () => {
      posApi();
      renderApp('/vender');

      const done = await sellPipe();
      expect(printed()).toHaveAttribute('data-paper', '80');
      await user().click(done.getByRole('combobox', { name: 'Papel' }));
      await user().click(await screen.findByRole('option', { name: '58 mm' }));

      expect(printed()).toHaveAttribute('data-paper', '58');
      expect(localStorage.getItem('cuadra.pos.paper')).toBe('58');
    });

    it('lee el ancho guardado en el equipo', async () => {
      localStorage.setItem('cuadra.pos.paper', '58');
      posApi();
      renderApp('/vender');

      const done = await sellPipe();

      expect(done.getByRole('combobox', { name: 'Papel' })).toHaveTextContent('58 mm');
      expect(printed()).toHaveAttribute('data-paper', '58');
    });
  });

  describe('atajos', () => {
    it('F2 enfoca la búsqueda', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await user().click((await findCartLine(/Tubo PVC/)).getByLabelText(/^Cantidad/));

      await user().keyboard('{F2}');

      expect(await searchBox()).toHaveFocus();
    });

    it('F4 no abre el cobro con el carrito vacío', async () => {
      posApi();
      renderApp('/vender');
      await searchBox();

      await user().keyboard('{F4}');

      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    });

    it('Esc con texto en la búsqueda solo la limpia', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      await user().type(await searchBox(), 'cem');

      await user().keyboard('{Escape}');

      expect(await searchBox()).toHaveValue('');
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
      expect(cartLine(/Tubo PVC/)).toBeDefined();
    });

    it('Esc pregunta y solo un clic en «Cancelar venta» la borra', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{Escape}');
      const dialog = within(await screen.findByRole('dialog', { name: '¿Cancelar la venta?' }));
      // The safe choice has the focus: a stray Enter keeps the sale.
      expect(dialog.getByRole('button', { name: 'Seguir vendiendo' })).toHaveFocus();
      await user().click(dialog.getByRole('button', { name: 'Cancelar venta' }));

      await waitFor(() => expect(screen.queryByRole('listitem', { name: /Tubo PVC/ })).not.toBeInTheDocument());
      expect(await searchBox()).toHaveFocus();
    });

    it('un escaneo después de Esc no borra la venta', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{Escape}');
      await screen.findByRole('dialog', { name: '¿Cancelar la venta?' });
      // The reader types the code and its Enter while the question is open.
      await user().keyboard('7701234567890{Enter}');

      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(cartLine(/Tubo PVC/)).toBeDefined();
      expect(await searchBox()).toHaveFocus();
    });

    it('Esc en la confirmación conserva la venta', async () => {
      posApi();
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);

      await user().keyboard('{Escape}');
      await screen.findByRole('dialog', { name: '¿Cancelar la venta?' });
      await user().keyboard('{Escape}');

      await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
      expect(cartLine(/Tubo PVC/)).toBeDefined();
    });
  });

  describe('descuento', () => {
    it('un cajero con tope 0 no ve el campo de descuento', async () => {
      posApi({ limit: { unlimited: false, max: '0' } });
      renderApp('/vender');
      await scan('TUB-PVC-12');

      await findCartLine(/Tubo PVC/);

      expect(screen.queryByLabelText(/^Descuento/)).not.toBeInTheDocument();
    });

    it('un cajero con tope 10 % no puede dar 15 %', async () => {
      posApi({ limit: { unlimited: false, max: '0.1' } });
      renderApp('/vender');
      await scan('TUB-PVC-12');
      const discount = (await findCartLine(/Tubo PVC/)).getByLabelText(/^Descuento/);

      await user().type(discount, '15');
      expect(discount).toHaveAccessibleDescription(/Máximo 10 %/);
      expect(screen.getByRole('button', { name: 'Cobrar' })).toBeDisabled();

      await user().clear(discount);
      await user().type(discount, '10');
      // 10.000 − 1.000 = 9.000 + IVA 1.710.
      expect(summaryValue('Total')).toBe('$ 10.710');
      expect(screen.getByRole('button', { name: 'Cobrar' })).toBeEnabled();
    });

    it('un administrador descuenta sin tope y el monto viaja antes de IVA', async () => {
      const api = posApi({ user: aUser('admin'), limit: { unlimited: true, max: '1' } });
      renderApp('/vender');
      await scan('TUB-PVC-12');
      const discount = (await findCartLine(/Tubo PVC/)).getByLabelText(/^Descuento/);

      await user().type(discount, '50');

      expect(summaryValue('Total')).toBe('$ 5.950');
      expect(summaryValue('Descuentos')).toBe('$ 5.000');
      await user().keyboard('{F4}');
      await checkout();
      await user().keyboard('{Enter}');
      await screen.findByRole('dialog', { name: 'Venta N.º 15' });
      expect((await posted(api)).lines).toEqual([
        { product_id: ids.pipe, quantity: '1', discount: '5000.00' },
      ]);
    });
  });

  describe('sede', () => {
    it('con varias sedes, pide elegir una antes de cobrar y la recuerda', async () => {
      posApi({ locations: [aLocation(), aLocation({ id: ids.north, name: 'Sede norte' })] });
      renderApp('/vender');
      await scan('TUB-PVC-12');
      await findCartLine(/Tubo PVC/);
      expect(screen.getByRole('button', { name: 'Cobrar' })).toBeDisabled();
      expect(screen.getByText('Elija la sede para cobrar.')).toBeInTheDocument();

      await user().click(screen.getByRole('combobox', { name: 'Sede' }));
      await user().click(await screen.findByRole('option', { name: 'Sede norte' }));
      expect(screen.getByRole('button', { name: 'Cobrar' })).toBeEnabled();

      cleanup();
      renderApp('/vender');
      expect(await screen.findByRole('combobox', { name: 'Sede' })).toHaveTextContent('Sede norte');
    });

    it('sin sedes activas explica qué hacer', async () => {
      posApi({ locations: [aLocation({ active: false })] });
      renderApp('/vender');

      expect(await screen.findByText('No hay sedes activas para vender')).toBeInTheDocument();
    });
  });

  it('conserva la venta en curso al salir y volver a la pantalla', async () => {
    posApi();
    renderApp('/vender');
    await scan('TUB-PVC-12');
    await findCartLine(/Tubo PVC/);

    cleanup();
    renderApp('/vender');

    expect(await findCartLine(/Tubo PVC/)).toBeDefined();
  });
});

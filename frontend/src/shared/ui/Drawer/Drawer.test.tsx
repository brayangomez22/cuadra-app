import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { IconButton } from '../IconButton';
import { IconMenu } from '../icons';
import { Drawer } from './Drawer';

function renderDrawer() {
  render(
    <Drawer title="Menú" trigger={<IconButton label="Abrir menú" icon={<IconMenu />} />}>
      <a href="/productos">Productos</a>
      <a href="/inventario">Inventario</a>
    </Drawer>,
  );
  return { trigger: screen.getByRole('button', { name: 'Abrir menú' }) };
}

describe('Drawer', () => {
  it('se abre con su botón y tiene nombre accesible', async () => {
    const { trigger } = renderDrawer();

    await userEvent.click(trigger);

    expect(screen.getByRole('dialog', { name: 'Menú' })).toBeInTheDocument();
    expect(trigger).toHaveAttribute('aria-expanded', 'true');
  });

  it('atrapa el foco: Tab desde el último elemento vuelve al primero', async () => {
    const { trigger } = renderDrawer();
    await userEvent.click(trigger);
    const focusables = Array.from(screen.getByRole('dialog').querySelectorAll<HTMLElement>('button, a'));
    const first = focusables[0];
    const last = focusables[focusables.length - 1];

    last?.focus();
    await userEvent.tab();

    expect(document.activeElement).toBe(first);
  });

  it('cierra con Escape y devuelve el foco al botón que lo abrió', async () => {
    const { trigger } = renderDrawer();
    await userEvent.click(trigger);

    await userEvent.keyboard('{Escape}');

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('cierra con el botón Cerrar', async () => {
    const { trigger } = renderDrawer();
    await userEvent.click(trigger);

    await userEvent.click(screen.getByRole('button', { name: 'Cerrar menú' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});

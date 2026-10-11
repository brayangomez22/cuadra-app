import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from '../Button';
import { Dialog, DialogClose } from './Dialog';

function renderDialog() {
  render(
    <Dialog
      trigger={<Button>Anular venta</Button>}
      title="¿Anular la venta?"
      description="La venta queda anulada y el stock vuelve a la bodega."
      footer={
        <>
          <DialogClose asChild>
            <Button variant="secondary">Cancelar</Button>
          </DialogClose>
          <Button variant="danger">Anular</Button>
        </>
      }
    >
      <label>
        Motivo
        <input />
      </label>
    </Dialog>,
  );
  return { trigger: screen.getByRole('button', { name: 'Anular venta' }) };
}

describe('Dialog', () => {
  it('mueve el foco dentro del diálogo al abrirse', async () => {
    const { trigger } = renderDialog();

    await userEvent.click(trigger);

    expect(screen.getByRole('dialog')).toContainElement(document.activeElement as HTMLElement);
  });

  it('atrapa el foco: Tab desde el último elemento vuelve al primero', async () => {
    const { trigger } = renderDialog();
    await userEvent.click(trigger);
    const dialog = screen.getByRole('dialog');
    const focusables = Array.from(dialog.querySelectorAll<HTMLElement>('button, input'));
    const first = focusables[0];
    const last = focusables[focusables.length - 1];

    last?.focus();
    await userEvent.tab();
    expect(document.activeElement).toBe(first);

    await userEvent.tab({ shift: true });
    expect(document.activeElement).toBe(last);
  });

  it('cierra con Escape y devuelve el foco al botón que lo abrió', async () => {
    const { trigger } = renderDialog();
    await userEvent.click(trigger);

    await userEvent.keyboard('{Escape}');

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(trigger).toHaveFocus();
  });

  it('tiene nombre y descripción accesibles', async () => {
    const { trigger } = renderDialog();
    await userEvent.click(trigger);

    const dialog = screen.getByRole('dialog', { name: '¿Anular la venta?' });
    expect(dialog).toHaveAccessibleDescription('La venta queda anulada y el stock vuelve a la bodega.');
  });

  it('cierra con el botón Cerrar', async () => {
    const { trigger } = renderDialog();
    await userEvent.click(trigger);

    await userEvent.click(screen.getByRole('button', { name: 'Cerrar' }));

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });
});

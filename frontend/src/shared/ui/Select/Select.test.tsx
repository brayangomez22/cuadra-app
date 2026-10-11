import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Select } from './Select';

const units = [
  { value: 'und', label: 'Unidad' },
  { value: 'm', label: 'Metro' },
  { value: 'kg', label: 'Kilogramo' },
];

describe('Select', () => {
  it('abre las opciones con el teclado y selecciona una', async () => {
    const onValueChange = vi.fn();
    render(<Select aria-label="Unidad" options={units} placeholder="Elegir" onValueChange={onValueChange} />);
    const trigger = screen.getByRole('combobox', { name: 'Unidad' });

    trigger.focus();
    await userEvent.keyboard('{Enter}');
    expect(await screen.findByRole('listbox')).toBeInTheDocument();
    await userEvent.keyboard('{ArrowDown}{Enter}');

    expect(onValueChange).toHaveBeenCalledWith('m');
    expect(trigger).toHaveTextContent('Metro');
  });
});

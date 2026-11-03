import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { SegmentedControl } from './SegmentedControl';

const options = [
  { value: 'out', label: 'Sale' },
  { value: 'in', label: 'Entra' },
];

describe('SegmentedControl', () => {
  it('es un grupo de radios con su nombre y se elige con un toque', async () => {
    const onChange = vi.fn();
    render(<SegmentedControl legend="Movimiento" name="direction" options={options} onChange={onChange} />);

    const group = screen.getByRole('group', { name: 'Movimiento' });
    expect(group).toBeInTheDocument();
    await userEvent.click(screen.getByRole('radio', { name: 'Entra' }));

    expect(screen.getByRole('radio', { name: 'Entra' })).toBeChecked();
    expect(onChange).toHaveBeenCalledTimes(1);
  });

  it('dice el error y marca el grupo como inválido', () => {
    render(<SegmentedControl legend="Movimiento" name="direction" options={options} error="Elija si sale o entra" />);

    const group = screen.getByRole('group', { name: 'Movimiento' });
    expect(group).toHaveAccessibleDescription('Elija si sale o entra');
    expect(screen.getByRole('radio', { name: 'Sale' })).toHaveAttribute('aria-invalid', 'true');
  });
});

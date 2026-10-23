import { render, screen } from '@testing-library/react';
import { Inline } from '../Inline';
import { Stack } from './Stack';

describe('Stack', () => {
  it('aplica el gap y la alineación pedidos', () => {
    render(
      <Stack gap="6" align="start" data-testid="stack">
        <p>Uno</p>
      </Stack>,
    );

    const stack = screen.getByTestId('stack');
    expect(stack).toHaveAttribute('data-gap', '6');
    expect(stack).toHaveAttribute('data-align', 'start');
  });

  it('usa el gap 4 por defecto y renderiza el elemento de as', () => {
    render(
      <Stack as="ul" aria-label="Pasos">
        <li>Uno</li>
      </Stack>,
    );

    expect(screen.getByRole('list', { name: 'Pasos' })).toHaveAttribute('data-gap', '4');
  });
});

describe('Inline', () => {
  it('aplica el gap, la justificación y el ajuste de línea', () => {
    render(
      <Inline gap="2" justify="between" wrap data-testid="inline">
        <span>Uno</span>
      </Inline>,
    );

    const inline = screen.getByTestId('inline');
    expect(inline).toHaveAttribute('data-gap', '2');
    expect(inline).toHaveAttribute('data-justify', 'between');
    expect(inline).toHaveAttribute('data-wrap', 'true');
  });
});

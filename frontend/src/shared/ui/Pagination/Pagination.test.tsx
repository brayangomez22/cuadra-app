import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Pagination } from './Pagination';

describe('Pagination', () => {
  it('muestra el rango visible y el total con separador de miles', () => {
    render(<Pagination label="Páginas de productos" total={1340} limit={20} offset={40} onOffsetChange={() => {}} />);

    const nav = screen.getByRole('navigation', { name: 'Páginas de productos' });
    expect(nav).toHaveTextContent('41–60 de 1.340');
  });

  it('deshabilita Anterior en la primera página y Siguiente en la última', () => {
    const { rerender } = render(
      <Pagination label="Páginas" total={45} limit={20} offset={0} onOffsetChange={() => {}} />,
    );
    expect(screen.getByRole('button', { name: 'Anterior' })).toBeDisabled();
    expect(screen.getByRole('button', { name: 'Siguiente' })).toBeEnabled();

    rerender(<Pagination label="Páginas" total={45} limit={20} offset={40} onOffsetChange={() => {}} />);
    expect(screen.getByText('41–45 de 45')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Anterior' })).toBeEnabled();
    expect(screen.getByRole('button', { name: 'Siguiente' })).toBeDisabled();
  });

  it('pide el desplazamiento de la página siguiente y de la anterior', async () => {
    const onOffsetChange = vi.fn();
    render(<Pagination label="Páginas" total={100} limit={20} offset={20} onOffsetChange={onOffsetChange} />);

    await userEvent.click(screen.getByRole('button', { name: 'Siguiente' }));
    await userEvent.click(screen.getByRole('button', { name: 'Anterior' }));

    expect(onOffsetChange.mock.calls).toEqual([[40], [0]]);
  });

  it('no muestra los botones si todo cabe en una página', () => {
    render(<Pagination label="Páginas" total={3} limit={20} offset={0} onOffsetChange={() => {}} />);

    expect(screen.getByText('1–3 de 3')).toBeInTheDocument();
    expect(screen.queryByRole('button')).not.toBeInTheDocument();
  });

  it('no muestra nada sin resultados', () => {
    const { container } = render(
      <Pagination label="Páginas" total={0} limit={20} offset={0} onOffsetChange={() => {}} />,
    );
    expect(container).toBeEmptyDOMElement();
  });
});

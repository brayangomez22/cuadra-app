import { render, screen } from '@testing-library/react';
import { Table, TableBody, TableCell, TableHead, TableHeaderCell, TableRow } from './Table';

function renderTable() {
  render(
    <Table caption="Existencias por sede">
      <TableHead>
        <TableRow>
          <TableHeaderCell>Producto</TableHeaderCell>
          <TableHeaderCell numeric>Cantidad</TableHeaderCell>
        </TableRow>
      </TableHead>
      <TableBody>
        <TableRow selected>
          <TableCell>Cemento gris 50kg</TableCell>
          <TableCell numeric>12</TableCell>
        </TableRow>
        <TableRow>
          <TableCell>Cable 12 AWG</TableCell>
          <TableCell numeric>2,5</TableCell>
        </TableRow>
      </TableBody>
    </Table>,
  );
}

describe('Table', () => {
  it('tiene nombre accesible desde su caption', () => {
    renderTable();

    expect(screen.getByRole('table', { name: 'Existencias por sede' })).toBeInTheDocument();
  });

  it('alinea a la derecha las celdas numéricas', () => {
    renderTable();

    expect(screen.getByRole('columnheader', { name: 'Cantidad' })).toHaveAttribute('data-numeric', 'true');
    expect(screen.getByRole('cell', { name: '12' })).toHaveAttribute('data-numeric', 'true');
    expect(screen.getByRole('cell', { name: 'Cemento gris 50kg' })).not.toHaveAttribute('data-numeric');
  });

  it('marca la fila seleccionada con aria-selected', () => {
    renderTable();
    const [, selected, other] = screen.getAllByRole('row');

    expect(selected).toHaveAttribute('aria-selected', 'true');
    expect(other).not.toHaveAttribute('aria-selected');
  });
});

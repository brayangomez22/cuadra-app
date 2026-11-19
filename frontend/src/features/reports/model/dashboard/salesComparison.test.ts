import { compareSales } from './salesComparison';

const totals = (total: string, sales: number) => ({ sales_count: sales, net: '0.00', tax: '0.00', total });

describe('compareSales', () => {
  it.each([
    {
      name: 'una ferretería que vende más que el mismo día de la semana pasada',
      today: '1200000.00',
      previous: '1000000.00',
      change: 'up',
      percent: '20',
    },
    {
      name: 'un almacén de ropa que vende menos',
      today: '800000.00',
      previous: '1000000.00',
      change: 'down',
      percent: '20',
    },
    { name: 'lo mismo que la semana pasada', today: '500000.00', previous: '500000.00', change: 'same', percent: '0' },
    {
      name: 'redondea el porcentaje hacia arriba desde la mitad',
      today: '1005000.00',
      previous: '1000000.00',
      change: 'up',
      percent: '1',
    },
    {
      name: 'sin ventas la semana pasada no hay porcentaje',
      today: '300000.00',
      previous: '0.00',
      change: 'up',
      percent: null,
    },
    { name: 'sin ventas en ninguno de los dos días', today: '0.00', previous: '0.00', change: 'same', percent: null },
  ])('$name', ({ today, previous, change, percent }) => {
    const result = compareSales(totals(today, 3), totals(previous, 4));
    expect(result.total).toBe(today);
    expect(result.previous).toBe(previous);
    expect(result.change).toBe(change);
    expect(result.percent).toBe(percent);
  });

  it.each([
    { name: 'divide el total entre el número de ventas', total: '1200000.00', sales: 10, ticket: '120000.00' },
    { name: 'redondea el ticket a centavos', total: '100000.00', sales: 3, ticket: '33333.33' },
    { name: 'sin ventas no hay ticket promedio', total: '0.00', sales: 0, ticket: null },
  ])('ticket promedio: $name', ({ total, sales, ticket }) => {
    const result = compareSales(totals(total, sales), totals('0.00', 0));
    expect(result.sales).toBe(sales);
    expect(result.averageTicket).toBe(ticket);
  });
});

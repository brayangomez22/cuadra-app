import { describe, expect, it } from 'vitest';
import { aCement, aGarment, aProduct } from '@/test/fixtures';
import { adjustedMessage, insufficientStockMessage, transferredMessage } from './stockChangeMessages';

const pipe = { product: aProduct(), variant: aProduct().variants[0]! };
const garment = { product: aGarment(), variant: aGarment().variants[0]! };
const cement = { product: aCement(), variant: aCement().variants[0]! };

describe('adjustedMessage', () => {
  it('dice lo que salió y con cuánto queda la sede', () => {
    const text = adjustedMessage({ item: pipe, delta: '-3', balanceAfter: '117', locationName: 'Bodega principal' });
    expect(text).toBe('Salen 3 und de Tubo PVC presión 1/2 pulgada. Bodega principal queda con 117 und.');
  });

  it('dice lo que entró, con la variante de una prenda', () => {
    const text = adjustedMessage({ item: garment, delta: '2', balanceAfter: '6', locationName: 'Sede norte' });
    expect(text).toMatch(/^Entran 2 und de Camiseta.*Sede norte queda con 6 und\.$/);
  });
});

describe('transferredMessage', () => {
  it('dice con cuánto queda cada sede', () => {
    const text = transferredMessage({
      item: cement,
      quantity: '2.5',
      from: { name: 'Bodega principal', balance: '10' },
      to: { name: 'Sede norte', balance: '4.5' },
    });
    expect(text).toBe(
      '2,5 bulto de Cemento gris 50kg. Bodega principal queda con 10 bulto y Sede norte con 4,5 bulto.',
    );
  });
});

describe('insufficientStockMessage', () => {
  it('nombra el producto y la sede', () => {
    expect(insufficientStockMessage(pipe, 'Sede norte')).toBe(
      'No hay stock suficiente de Tubo PVC presión 1/2 pulgada en Sede norte.',
    );
  });
});

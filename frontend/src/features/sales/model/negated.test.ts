import { negated } from './negated';

describe('negated', () => {
  it('pone el signo a lo que sale del cajón', () => {
    expect(negated('12000.00')).toBe('-12000.00');
  });

  it('cero queda en cero, sin signo', () => {
    expect(negated('0.00')).toBe('0.00');
  });
});

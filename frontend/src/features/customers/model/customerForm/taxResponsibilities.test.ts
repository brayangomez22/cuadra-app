import type { TaxResponsibility } from '../../api/types';
import { toggleResponsibility } from './taxResponsibilities';

describe('toggleResponsibility', () => {
  it.each<{
    name: string;
    current: TaxResponsibility[];
    value: TaxResponsibility;
    checked: boolean;
    want: TaxResponsibility[];
  }>([
    { name: 'marcar otra quita «No responsable»', current: ['R-99-PN'], value: 'O-13', checked: true, want: ['O-13'] },
    {
      name: 'marcar «No responsable» deja solo esa',
      current: ['O-13', 'O-15'],
      value: 'R-99-PN',
      checked: true,
      want: ['R-99-PN'],
    },
    { name: 'conserva el orden del RUT', current: ['O-47'], value: 'O-15', checked: true, want: ['O-15', 'O-47'] },
    { name: 'desmarcar quita solo esa', current: ['O-13', 'O-15'], value: 'O-13', checked: false, want: ['O-15'] },
  ])('$name', ({ current, value, checked, want }) => {
    expect(toggleResponsibility(current, value, checked)).toEqual(want);
  });
});

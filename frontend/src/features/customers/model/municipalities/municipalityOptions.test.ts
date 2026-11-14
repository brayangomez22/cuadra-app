import { municipalities } from '@/test/fixtures';
import { departmentOf, departmentOptions, municipalityOptions } from './municipalityOptions';

describe('municipalityOptions', () => {
  it('lista cada departamento una vez, en el orden del catálogo', () => {
    expect(departmentOptions(municipalities)).toEqual([
      { value: '05', label: 'Antioquia' },
      { value: '11', label: 'Bogotá, D.C.' },
    ]);
  });

  it('lista los municipios del departamento elegido (ferretería de Envigado, almacén de ropa de Medellín)', () => {
    expect(municipalityOptions(municipalities, '05')).toEqual([
      { value: '05266', label: 'Envigado' },
      { value: '05001', label: 'Medellín' },
    ]);
  });

  it('sin departamento no hay municipios', () => {
    expect(municipalityOptions(municipalities, '')).toEqual([]);
  });

  it.each([
    { code: '05001', want: '05' },
    { code: '11001', want: '11' },
    { code: '', want: '' },
  ])('el departamento de $code es «$want»', ({ code, want }) => {
    expect(departmentOf(code)).toBe(want);
  });
});

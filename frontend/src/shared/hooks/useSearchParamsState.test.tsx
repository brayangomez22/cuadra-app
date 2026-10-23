import { act, renderHook } from '@testing-library/react';
import type { ReactNode } from 'react';
import { MemoryRouter, useLocation } from 'react-router';
import { enumParam, pageParam, useSearchParamsState, type SearchParamCodec } from './useSearchParamsState';

function setup<T>(url: string, param: { key: string; codec: SearchParamCodec<T>; reset?: string[] }) {
  const wrapper = ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={[url]}>{children}</MemoryRouter>
  );
  return renderHook(
    () => {
      const [value, setValue] = useSearchParamsState(param.key, param.codec, { reset: param.reset });
      return { value, setValue, search: useLocation().search };
    },
    { wrapper },
  );
}

describe('useSearchParamsState', () => {
  it('lee y escribe la página como número', () => {
    const { result } = setup('/productos?pagina=3&q=tubo', { key: 'pagina', codec: pageParam });

    expect(result.current.value).toBe(3);

    act(() => result.current.setValue(4));

    expect(result.current.value).toBe(4);
    expect(result.current.search).toBe('?pagina=4&q=tubo');
  });

  it('quita el parámetro al volver al valor por defecto', () => {
    const { result } = setup('/productos?q=tubo&pagina=2', { key: 'pagina', codec: pageParam });

    act(() => result.current.setValue(1));

    expect(result.current.value).toBe(1);
    expect(result.current.search).toBe('?q=tubo');
  });

  it.each([['abc'], ['0'], ['-2'], ['1.5']])('un valor inválido en la URL (%s) cae al valor por defecto', (raw) => {
    const { result } = setup(`/productos?pagina=${raw}`, { key: 'pagina', codec: pageParam });

    expect(result.current.value).toBe(1);
  });

  it('un enum solo acepta sus valores', () => {
    const status = enumParam(['activos', 'inactivos', 'todos'], 'activos');

    expect(setup('/productos?estado=todos', { key: 'estado', codec: status }).result.current.value).toBe('todos');
    expect(setup('/productos?estado=otro', { key: 'estado', codec: status }).result.current.value).toBe('activos');
  });

  it('reset borra los parámetros indicados al cambiar el valor', () => {
    const status = enumParam(['activos', 'todos'], 'activos');
    const { result } = setup('/productos?pagina=5&q=tubo', { key: 'estado', codec: status, reset: ['pagina'] });

    act(() => result.current.setValue('todos'));

    expect(result.current.search).toBe('?q=tubo&estado=todos');
  });
});

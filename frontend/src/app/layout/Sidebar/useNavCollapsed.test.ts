import { act, renderHook } from '@testing-library/react';
import { useNavCollapsed } from './useNavCollapsed';

describe('useNavCollapsed', () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it('empieza con el menú abierto en un equipo nuevo', () => {
    const { result } = renderHook(() => useNavCollapsed());

    expect(result.current.collapsed).toBe(false);
  });

  it('guarda en el equipo cómo quedó el menú', () => {
    const first = renderHook(() => useNavCollapsed());
    act(() => {
      first.result.current.toggle();
    });
    expect(first.result.current.collapsed).toBe(true);

    expect(renderHook(() => useNavCollapsed()).result.current.collapsed).toBe(true);
  });

  it('empieza abierto si el navegador no deja leer lo guardado', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('bloqueado');
    });

    expect(renderHook(() => useNavCollapsed()).result.current.collapsed).toBe(false);
    vi.restoreAllMocks();
  });
});

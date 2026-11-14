import { act, renderHook } from '@testing-library/react';
import { useTheme } from './useTheme';

const root = document.documentElement;

describe('useTheme', () => {
  beforeEach(() => {
    localStorage.clear();
    delete root.dataset.theme;
  });

  it('sigue al sistema en un equipo nuevo', () => {
    const { result } = renderHook(() => useTheme());

    expect(result.current.theme).toBe('system');
    expect(root.dataset.theme).toBeUndefined();
  });

  it('pone el tema elegido en la página y lo guarda en el equipo', () => {
    const first = renderHook(() => useTheme());
    act(() => {
      first.result.current.setTheme('dark');
    });

    expect(root.dataset.theme).toBe('dark');
    expect(renderHook(() => useTheme()).result.current.theme).toBe('dark');
  });

  it('dos controles en pantalla muestran el mismo tema', () => {
    const one = renderHook(() => useTheme());
    const other = renderHook(() => useTheme());
    act(() => {
      one.result.current.setTheme('light');
    });

    expect(other.result.current.theme).toBe('light');
  });

  it('sigue al sistema si lo guardado no es un tema', () => {
    localStorage.setItem('cuadra.theme', 'azul');

    expect(renderHook(() => useTheme()).result.current.theme).toBe('system');
  });

  it('sigue al sistema si el navegador no deja leer lo guardado', () => {
    vi.spyOn(Storage.prototype, 'getItem').mockImplementation(() => {
      throw new Error('bloqueado');
    });

    expect(renderHook(() => useTheme()).result.current.theme).toBe('system');
    vi.restoreAllMocks();
  });
});

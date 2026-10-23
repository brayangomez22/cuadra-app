import { act, renderHook } from '@testing-library/react';
import { useAddedMark } from './useAddedMark';

beforeEach(() => vi.useFakeTimers());
afterEach(() => vi.useRealTimers());

describe('useAddedMark', () => {
  it('marca la línea agregada y la marca se apaga sola a los 1,5 s', () => {
    const { result } = renderHook(() => useAddedMark());
    act(() => result.current.mark('tubo'));
    expect(result.current.added).toEqual({ key: 'tubo', seq: 1 });

    act(() => {
      vi.advanceTimersByTime(1499);
    });
    expect(result.current.added?.key).toBe('tubo');
    act(() => {
      vi.advanceTimersByTime(1);
    });
    expect(result.current.added).toBeUndefined();
  });

  it('marcar la misma línea otra vez la apaga un cuadro y la reinicia', () => {
    const { result } = renderHook(() => useAddedMark());
    act(() => result.current.mark('camiseta'));
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    act(() => result.current.mark('camiseta'));
    expect(result.current.added).toBeUndefined();

    act(() => {
      vi.advanceTimersByTime(16);
    });
    expect(result.current.added).toEqual({ key: 'camiseta', seq: 2 });
    act(() => {
      vi.advanceTimersByTime(1000);
    });
    expect(result.current.added?.key).toBe('camiseta');
  });

  it('marcar otra línea la marca de una vez', () => {
    const { result } = renderHook(() => useAddedMark());
    act(() => result.current.mark('tubo'));
    act(() => result.current.mark('camiseta'));
    expect(result.current.added).toEqual({ key: 'camiseta', seq: 2 });
  });
});

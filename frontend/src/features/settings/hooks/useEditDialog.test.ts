import { act, renderHook } from '@testing-library/react';
import { unitOf } from '@/test/fixtures';
import { useEditDialog } from './useEditDialog';

describe('useEditDialog', () => {
  it('empieza cerrado', () => {
    const { result } = renderHook(() => useEditDialog());
    expect(result.current.isOpen).toBe(false);
    expect(result.current.editing).toBeNull();
  });

  it('abre para agregar, sin elemento', () => {
    const { result } = renderHook(() => useEditDialog());
    act(() => result.current.openNew());
    expect(result.current.isOpen).toBe(true);
    expect(result.current.editing).toBeNull();
  });

  it('abre para editar con el elemento elegido', () => {
    const pair = unitOf('par');
    const { result } = renderHook(() => useEditDialog<typeof pair>());
    act(() => result.current.openEdit(pair));
    expect(result.current.isOpen).toBe(true);
    expect(result.current.editing).toBe(pair);
  });

  it('cierra y olvida el elemento', () => {
    const { result } = renderHook(() => useEditDialog());
    act(() => result.current.openEdit(unitOf('kg')));
    act(() => result.current.close());
    expect(result.current.isOpen).toBe(false);
    expect(result.current.editing).toBeNull();
  });
});

import { act, renderHook } from '@testing-library/react';
import { axes, unitOf } from '@/test/fixtures';
import { useDeleteConfirmation } from './useDeleteConfirmation';

const meter = unitOf('m');
const [size] = axes;

describe('useDeleteConfirmation', () => {
  it('empieza sin nada que confirmar', () => {
    const { result } = renderHook(() => useDeleteConfirmation(vi.fn()));
    expect(result.current.target).toBeNull();
    expect(result.current.dialogProps.open).toBe(false);
  });

  it('pide confirmar el elemento elegido', () => {
    const { result } = renderHook(() => useDeleteConfirmation(vi.fn()));
    act(() => result.current.ask(meter));
    expect(result.current.target).toBe(meter);
    expect(result.current.dialogProps.open).toBe(true);
  });

  it('al confirmar borra ese elemento', async () => {
    const remove = vi.fn(() => Promise.resolve());
    const { result } = renderHook(() => useDeleteConfirmation(remove));
    act(() => result.current.ask(size!));
    await act(() => result.current.dialogProps.onConfirm());
    expect(remove).toHaveBeenCalledWith(size);
  });

  it('cerrar el diálogo olvida el elemento', () => {
    const { result } = renderHook(() => useDeleteConfirmation(vi.fn()));
    act(() => result.current.ask(meter));
    act(() => result.current.dialogProps.onOpenChange(false));
    expect(result.current.target).toBeNull();
  });

  it('si el borrado falla, el error llega al diálogo y el elemento sigue elegido', async () => {
    const failure = new Error('La unidad está en uso');
    const { result } = renderHook(() => useDeleteConfirmation(() => Promise.reject(failure)));
    act(() => result.current.ask(meter));
    await expect(result.current.dialogProps.onConfirm()).rejects.toBe(failure);
    expect(result.current.target).toBe(meter);
  });
});

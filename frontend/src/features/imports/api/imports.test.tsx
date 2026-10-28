import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { act, renderHook, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { fakeApi, json } from '@/test/fakeApi';
import { anImport, counts, importId } from '../testing/importApi';
import { POLL_MS, useImport } from './imports';

function setup() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const invalidated = vi.spyOn(queryClient, 'invalidateQueries');
  const wrapper = ({ children }: { children: ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
  return { invalidated, wrapper };
}

const keysOf = (spy: ReturnType<typeof setup>['invalidated']) => spy.mock.calls.map(([filters]) => filters?.queryKey);

describe('useImport', () => {
  afterEach(() => vi.useRealTimers());

  it('mientras corre vuelve a preguntar y, al terminar, el catálogo y el inventario quedan viejos', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    let call = 0;
    const running = anImport({ status: 'running', counts: counts({ created: 1 }) });
    const done = anImport({ status: 'completed', counts: counts({ created: 2 }) });
    fakeApi({ 'GET /api/v1/imports/{importId}': () => json(200, call++ === 0 ? running : done) });
    const { invalidated, wrapper } = setup();

    const { result } = renderHook(() => useImport(importId), { wrapper });
    await waitFor(() => expect(result.current.data?.status).toBe('running'));
    expect(invalidated).not.toHaveBeenCalled();

    await act(() => vi.advanceTimersByTimeAsync(POLL_MS));

    await waitFor(() => expect(result.current.data?.status).toBe('completed'));
    expect(keysOf(invalidated)).toEqual([['catalog'], ['inventory']]);
  });

  it('una importación que ya terminó no invalida nada ni vuelve a preguntar', async () => {
    vi.useFakeTimers({ shouldAdvanceTime: true });
    const api = fakeApi({ 'GET /api/v1/imports/{importId}': json(200, anImport({ status: 'failed' })) });
    const { invalidated, wrapper } = setup();

    const { result } = renderHook(() => useImport(importId), { wrapper });
    await waitFor(() => expect(result.current.data?.status).toBe('failed'));
    await act(() => vi.advanceTimersByTimeAsync(POLL_MS * 3));

    expect(api.calls(`GET /api/v1/imports/${importId}`)).toHaveLength(1);
    expect(invalidated).not.toHaveBeenCalled();
  });
});

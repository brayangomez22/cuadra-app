import { renderHook } from '@testing-library/react';
import { track } from '@/features/analytics';
import { aChecklist, hardwareSteps } from '../testing/checklists';
import { useStepCompletedEvents } from './useStepCompletedEvents';

vi.mock('@/features/analytics', () => ({ track: vi.fn() }));

const tenantId = '01920000-0000-7000-8000-0000000000aa';

afterEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
});

describe('useStepCompletedEvents', () => {
  it('envía el evento una vez por paso completado', () => {
    const { rerender } = renderHook(({ steps }) => useStepCompletedEvents(tenantId, steps), {
      initialProps: { steps: aChecklist(hardwareSteps, ['products']).steps },
    });
    expect(track).toHaveBeenCalledWith('getting_started_step_completed', { step: 'products' });

    rerender({ steps: aChecklist(hardwareSteps, ['products', 'first_sale']).steps });

    expect(track).toHaveBeenCalledTimes(2);
    expect(track).toHaveBeenLastCalledWith('getting_started_step_completed', { step: 'first_sale' });
  });

  it('no repite el evento al volver a abrir la app en el mismo navegador', () => {
    const steps = aChecklist(hardwareSteps, ['products']).steps;
    renderHook(() => useStepCompletedEvents(tenantId, steps)).unmount();
    renderHook(() => useStepCompletedEvents(tenantId, steps));

    expect(track).toHaveBeenCalledTimes(1);
  });

  it('sin la lista cargada no envía nada', () => {
    renderHook(() => useStepCompletedEvents(tenantId, undefined));

    expect(track).not.toHaveBeenCalled();
  });
});

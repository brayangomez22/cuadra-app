import { renderHook } from '@testing-library/react';
import { aUser } from '@/test/fakeApi';
import { useAuth, type AuthState } from './useAuth';
import { useModule } from './useModule';

vi.mock('./useAuth');

function withAuth(state: AuthState) {
  vi.mocked(useAuth).mockReturnValue({ ...state, startSession: vi.fn(), logout: vi.fn() });
}

describe('useModule', () => {
  it('es verdadero si el módulo está activo para la empresa', () => {
    // Almacén de ropa con variantes y unidades alternativas.
    withAuth({ status: 'authenticated', user: { ...aUser(), modules: ['variants', 'unit_conversions'] } });

    expect(renderHook(() => useModule('unit_conversions')).result.current).toBe(true);
  });

  it('es falso si el módulo no está activo', () => {
    // Ferretería sin variantes.
    withAuth({ status: 'authenticated', user: { ...aUser(), modules: ['unit_conversions'] } });

    expect(renderHook(() => useModule('variants')).result.current).toBe(false);
  });

  it('es falso sin sesión', () => {
    withAuth({ status: 'anonymous' });

    expect(renderHook(() => useModule('variants')).result.current).toBe(false);
  });
});

import { renderHook } from '@testing-library/react';
import { aUser } from '@/test/fakeApi';
import { useAuth, type AuthState } from './useAuth';
import { useFlag } from './useFlag';

vi.mock('./useAuth');

function withAuth(state: AuthState) {
  vi.mocked(useAuth).mockReturnValue({ ...state, startSession: vi.fn(), logout: vi.fn() });
}

describe('useFlag', () => {
  it('es verdadero si la flag está prendida para la empresa', () => {
    // Una ferretería piloto que ya ve el botón de comentarios.
    withAuth({ status: 'authenticated', user: { ...aUser(), flags: ['feedback'] } });

    expect(renderHook(() => useFlag('feedback')).result.current).toBe(true);
  });

  it('es falso si la flag está apagada', () => {
    // Un almacén de calzado que todavía no la tiene.
    withAuth({ status: 'authenticated', user: { ...aUser(), business_template: 'ropa_calzado', flags: [] } });

    expect(renderHook(() => useFlag('feedback')).result.current).toBe(false);
  });

  it('es falso sin sesión', () => {
    withAuth({ status: 'anonymous' });

    expect(renderHook(() => useFlag('feedback')).result.current).toBe(false);
  });
});

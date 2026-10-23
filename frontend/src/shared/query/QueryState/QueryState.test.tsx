import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { ApiError } from '@/shared/api/errors';
import { QueryState, type QueryStatus } from './QueryState';

function loading(): QueryStatus {
  return { isPending: true, isError: false, error: null, refetch: vi.fn() };
}

function failed(message: string): QueryStatus {
  const error = new ApiError(500, { error: { code: 'internal', message } });
  return { isPending: false, isError: true, error, refetch: vi.fn() };
}

function loaded(): QueryStatus {
  return { isPending: false, isError: false, error: null, refetch: vi.fn() };
}

describe('QueryState', () => {
  it('muestra el error de la query que falló aunque otra siga cargando', () => {
    render(
      <QueryState queries={[loading(), failed('La base de datos no responde')]} errorTitle="No pudimos cargar">
        <p>Contenido</p>
      </QueryState>,
    );

    expect(screen.getByRole('alert')).toHaveTextContent('No pudimos cargar');
    expect(screen.getByRole('alert')).toHaveTextContent('La base de datos no responde');
    expect(screen.queryByRole('status')).not.toBeInTheDocument();
    expect(screen.queryByText('Contenido')).not.toBeInTheDocument();
  });

  it('Reintentar vuelve a pedir solo la que falló', async () => {
    const ok = loaded();
    const bad = failed('Error');
    render(
      <QueryState queries={[ok, bad]} errorTitle="No pudimos cargar">
        <p>Contenido</p>
      </QueryState>,
    );

    await userEvent.click(screen.getByRole('button', { name: 'Reintentar' }));

    expect(bad.refetch).toHaveBeenCalledOnce();
    expect(ok.refetch).not.toHaveBeenCalled();
  });

  it('muestra el cargando mientras alguna query no llega', () => {
    render(
      <QueryState queries={[loaded(), loading()]} errorTitle="No pudimos cargar">
        <p>Contenido</p>
      </QueryState>,
    );

    expect(screen.getByRole('status')).toHaveTextContent('Cargando…');
    expect(screen.queryByText('Contenido')).not.toBeInTheDocument();
  });

  it('muestra el vacío si isEmpty', () => {
    render(
      <QueryState queries={[loaded()]} errorTitle="No pudimos cargar" isEmpty empty={<p>Aún no hay productos</p>}>
        <p>Contenido</p>
      </QueryState>,
    );

    expect(screen.getByText('Aún no hay productos')).toBeInTheDocument();
    expect(screen.queryByText('Contenido')).not.toBeInTheDocument();
  });

  it('muestra los hijos con datos, también como función', () => {
    render(
      <QueryState queries={[loaded()]} errorTitle="No pudimos cargar" isEmpty={false} empty={<p>Vacío</p>}>
        {() => <p>Contenido</p>}
      </QueryState>,
    );

    expect(screen.getByText('Contenido')).toBeInTheDocument();
  });
});

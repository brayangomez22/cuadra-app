import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { fakeCamera, noCamera } from '@/test/camera';
import { BarcodeScanner } from './BarcodeScanner';

describe('BarcodeScanner sin cámara', () => {
  it.each([
    { name: 'el usuario niega el permiso', error: 'NotAllowedError', title: 'No tenemos permiso para usar la cámara' },
    { name: 'el equipo no tiene cámara', error: 'NotFoundError', title: 'No encontramos una cámara en este equipo' },
    { name: 'otra aplicación usa la cámara', error: 'NotReadableError', title: 'No pudimos abrir la cámara' },
  ])('si $name muestra un mensaje claro y la alternativa de digitar el código', async ({ error, title }) => {
    fakeCamera({ error });
    const onScan = vi.fn();
    render(<BarcodeScanner onScan={onScan} />);

    expect(await screen.findByRole('alert')).toHaveTextContent(title);
    expect(screen.queryByText('Apunte la cámara al código')).not.toBeInTheDocument();
    await userEvent.type(screen.getByLabelText('Digitar el código'), '7701234567890{Enter}');
    expect(onScan).toHaveBeenCalledExactlyOnceWith('7701234567890');
  });

  it('fuera de HTTPS explica que la cámara necesita una conexión segura', async () => {
    noCamera({ secure: false });
    render(<BarcodeScanner onScan={vi.fn()} />);

    expect(await screen.findByRole('alert')).toHaveTextContent('La cámara necesita una conexión segura (HTTPS)');
    expect(screen.getByLabelText('Digitar el código')).toBeInTheDocument();
  });

  it('en un navegador sin acceso a cámaras lo explica', async () => {
    noCamera({ secure: true });
    render(<BarcodeScanner onScan={vi.fn()} />);

    expect(await screen.findByRole('alert')).toHaveTextContent('No encontramos una cámara en este equipo');
  });
});

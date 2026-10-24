import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { fakeCamera } from '@/test/camera';
import { BarcodeScanner } from './BarcodeScanner';

const user = () => userEvent.setup({ delay: null });

describe('BarcodeScanner', () => {
  it('una lectura dispara onScan una sola vez aunque el código siga en cámara', async () => {
    const camera = fakeCamera({ code: '7702057000123' });
    const onScan = vi.fn();
    render(<BarcodeScanner onScan={onScan} />);

    await waitFor(() => expect(onScan).toHaveBeenCalledWith('7702057000123'));
    const seen = camera.detect.mock.calls.length;
    await waitFor(() => expect(camera.detect.mock.calls.length).toBeGreaterThan(seen + 3));

    expect(onScan).toHaveBeenCalledTimes(1);
    expect(screen.getByRole('status')).toHaveTextContent('Código leído: 7702057000123');
  });

  it('lee el código que llega a la cámara después de abrirla', async () => {
    const camera = fakeCamera();
    const onScan = vi.fn();
    render(<BarcodeScanner onScan={onScan} />);

    expect(await screen.findByText('Apunte la cámara al código')).toBeInTheDocument();
    await waitFor(() => expect(camera.detect).toHaveBeenCalled());
    camera.show('CAM-AZ-M');

    await waitFor(() => expect(onScan).toHaveBeenCalledWith('CAM-AZ-M'));
  });

  it('pide la cámara trasera, sin audio', async () => {
    const camera = fakeCamera();
    render(<BarcodeScanner onScan={vi.fn()} />);

    await waitFor(() =>
      expect(camera.getUserMedia).toHaveBeenCalledWith({
        video: { facingMode: { ideal: 'environment' } },
        audio: false,
      }),
    );
  });

  it('digitar el código dispara onScan', async () => {
    fakeCamera();
    const onScan = vi.fn();
    render(<BarcodeScanner onScan={onScan} />);

    await user().type(screen.getByLabelText('Digitar el código'), ' 7701234567890 ');
    await user().click(screen.getByRole('button', { name: 'Usar el código' }));

    expect(onScan).toHaveBeenCalledExactlyOnceWith('7701234567890');
    expect(screen.getByLabelText('Digitar el código')).toHaveValue('');
  });

  it('no dispara onScan con el código digitado vacío', async () => {
    fakeCamera();
    const onScan = vi.fn();
    render(<BarcodeScanner onScan={onScan} />);

    await user().type(screen.getByLabelText('Digitar el código'), '   {Enter}');

    expect(onScan).not.toHaveBeenCalled();
  });

  it('muestra la linterna solo si la cámara la ofrece y la enciende', async () => {
    const camera = fakeCamera({ torch: true });
    render(<BarcodeScanner onScan={vi.fn()} />);

    const torch = await screen.findByRole('button', { name: 'Linterna' });
    expect(torch).toHaveAttribute('aria-pressed', 'false');
    await user().click(torch);

    expect(camera.track.applyConstraints).toHaveBeenCalledWith({ advanced: [{ torch: true }] });
    expect(torch).toHaveAttribute('aria-pressed', 'true');
  });

  it('sin linterna en la cámara no muestra el botón', async () => {
    fakeCamera();
    render(<BarcodeScanner onScan={vi.fn()} />);

    await screen.findByText('Apunte la cámara al código');
    expect(screen.queryByRole('button', { name: 'Linterna' })).not.toBeInTheDocument();
  });

  it('apaga la cámara al cerrarse', async () => {
    const camera = fakeCamera();
    const { unmount } = render(<BarcodeScanner onScan={vi.fn()} />);
    await waitFor(() => expect(camera.detect).toHaveBeenCalled());

    unmount();

    expect(camera.track.stop).toHaveBeenCalled();
    const calls = camera.detect.mock.calls.length;
    await new Promise((resolve) => setTimeout(resolve, 400));
    expect(camera.detect).toHaveBeenCalledTimes(calls);
  });
});

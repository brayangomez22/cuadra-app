import { render, waitFor } from '@testing-library/react';
import { fakeCamera } from '@/test/camera';
import { BarcodeScanner } from './BarcodeScanner';

// The library that reads the codes where the browser cannot: replaced by a
// detector that always sees the same code.
const fallback = vi.hoisted(() => ({
  detect: vi.fn(() => Promise.resolve([{ rawValue: '7702057000123' }])),
  formats: vi.fn(),
}));
vi.mock('barcode-detector/ponyfill', () => ({
  BarcodeDetector: class {
    constructor(options: unknown) {
      fallback.formats(options);
    }
    detect = fallback.detect;
  },
  prepareZXingModule: vi.fn(),
}));

describe('BarcodeScanner con el lector de respaldo', () => {
  it.each([
    { name: 'el navegador no tiene BarcodeDetector', formats: null },
    { name: 'el BarcodeDetector del navegador no lee todos los formatos', formats: ['qr_code'] },
  ])('lee con el respaldo cuando $name', async ({ formats }) => {
    fakeCamera({ formats });
    const onScan = vi.fn();
    render(<BarcodeScanner onScan={onScan} />);

    await waitFor(() => expect(onScan).toHaveBeenCalledWith('7702057000123'));
    expect(fallback.formats).toHaveBeenCalledWith({
      formats: ['ean_13', 'ean_8', 'code_128', 'upc_a', 'upc_e', 'qr_code'],
    });
  });
});

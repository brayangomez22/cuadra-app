import { BarcodeDetector, prepareZXingModule, type BarcodeFormat } from 'barcode-detector/ponyfill';
// zxing-wasm comes with barcode-detector, which pins its version: the WASM
// file always matches the library that loads it.
import wasmUrl from 'zxing-wasm/reader/zxing_reader.wasm?url';
import type { Detector } from './detector';

// The library fetches its WASM from a public CDN by default: it is served
// from our own origin instead, like the rest of the app.
prepareZXingModule({
  overrides: {
    locateFile: (path: string, prefix: string) => (path.endsWith('.wasm') ? wasmUrl : prefix + path),
  },
});

/** The BarcodeDetector of the barcode-detector library, on ZXing compiled to WASM. */
export function createFallbackDetector(formats: string[]): Detector {
  return new BarcodeDetector({ formats: formats as BarcodeFormat[] });
}

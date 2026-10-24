/** EAN-13, EAN-8, Code 128, UPC and QR: the codes printed on the products a business sells. */
export const FORMATS = ['ean_13', 'ean_8', 'code_128', 'upc_a', 'upc_e', 'qr_code'];

/** The part of the BarcodeDetector API the scanner uses, native or the fallback's. */
export type Detector = { detect: (source: HTMLVideoElement) => Promise<{ rawValue: string }[]> };

type DetectorClass = {
  new (options: { formats: string[] }): Detector;
  getSupportedFormats: () => Promise<readonly string[]>;
};

/**
 * The browser's BarcodeDetector when it reads every format (Chrome on
 * Android); else the fallback library, loaded only then (Safari, Firefox,
 * Chrome on a Windows PC), so the devices that do not need it never fetch it.
 */
export async function createDetector(): Promise<Detector> {
  const Native = (globalThis as { BarcodeDetector?: DetectorClass }).BarcodeDetector;
  if (Native) {
    const supported = await Native.getSupportedFormats();
    if (FORMATS.every((format) => supported.includes(format))) return new Native({ formats: FORMATS });
  }
  const { createFallbackDetector } = await import('./fallbackDetector');
  return createFallbackDetector(FORMATS);
}

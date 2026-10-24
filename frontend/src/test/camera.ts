import { onTestFinished } from 'vitest';

const ALL_FORMATS = ['ean_13', 'ean_8', 'code_128', 'upc_a', 'upc_e', 'qr_code'];

type Options = {
  /** The code in front of the camera at the start; null shows nothing. */
  code?: string | null;
  /** The name of the DOMException getUserMedia rejects with (NotAllowedError...). */
  error?: string;
  /** Whether the camera offers a flashlight. */
  torch?: boolean;
  /** The formats the browser's BarcodeDetector reads; null leaves the browser without one. */
  formats?: string[] | null;
};

/**
 * A camera and a BarcodeDetector for jsdom, which has neither: getUserMedia
 * answers a stream with one track, and detect sees the code the test puts in
 * front of the camera with show(). Everything is restored when the test ends.
 */
export function fakeCamera({ code = null, error, torch = false, formats = ALL_FORMATS }: Options = {}) {
  let inView = code;
  const track = {
    stop: vi.fn(),
    getCapabilities: () => (torch ? { torch: true } : {}),
    applyConstraints: vi.fn(() => Promise.resolve()),
  };
  const stream = { getTracks: () => [track], getVideoTracks: () => [track] };
  const getUserMedia = vi.fn(() =>
    error ? Promise.reject(new DOMException('La cámara falló', error)) : Promise.resolve(stream),
  );
  const detect = vi.fn(() => Promise.resolve(inView === null ? [] : [{ rawValue: inView, format: 'ean_13' }]));

  define(navigator, 'mediaDevices', { getUserMedia });
  const play = vi.spyOn(HTMLMediaElement.prototype, 'play').mockResolvedValue(undefined);
  onTestFinished(() => play.mockRestore());
  if (formats) {
    vi.stubGlobal(
      'BarcodeDetector',
      class {
        static getSupportedFormats = () => Promise.resolve(formats);
        detect = detect;
      },
    );
  }

  return {
    getUserMedia,
    detect,
    track,
    /** Puts a code in front of the camera, or takes it away with null. */
    show: (next: string | null) => {
      inView = next;
    },
  };
}

/** A browser without camera access at all, as outside HTTPS. */
export function noCamera({ secure }: { secure: boolean }) {
  define(window, 'isSecureContext', secure);
}

function define(target: object, key: string, value: unknown) {
  const had = Object.getOwnPropertyDescriptor(target, key);
  Object.defineProperty(target, key, { value, configurable: true, writable: true });
  onTestFinished(() => {
    if (had) Object.defineProperty(target, key, had);
    else Reflect.deleteProperty(target, key);
  });
}

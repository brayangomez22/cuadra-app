import type { CameraStatus } from './useCamera';

export type ScannerProblem = Exclude<CameraStatus, 'starting' | 'ready'> | 'reader';

/** What went wrong and what the cashier can do, always with the typed code as the way out. */
export const scannerProblems: Record<ScannerProblem, { title: string; description: string }> = {
  denied: {
    title: 'No tenemos permiso para usar la cámara',
    description: 'Permita la cámara para este sitio en la configuración del navegador, o digite el código abajo.',
  },
  no_camera: {
    title: 'No encontramos una cámara en este equipo',
    description: 'Digite el código abajo o use un lector de códigos de barras.',
  },
  insecure: {
    title: 'La cámara necesita una conexión segura (HTTPS)',
    description: 'Abra Cuadra con una dirección que empiece por https://, o digite el código abajo.',
  },
  failed: {
    title: 'No pudimos abrir la cámara',
    description:
      'Puede que otra aplicación la esté usando. Ciérrela y vuelva a abrir el escáner, o digite el código abajo.',
  },
  reader: {
    title: 'No pudimos iniciar el lector de códigos',
    description: 'Revise la conexión y vuelva a abrir el escáner, o digite el código abajo.',
  },
};

export function scannerProblem(status: CameraStatus, readerFailed: boolean): ScannerProblem | undefined {
  if (status === 'starting' || status === 'ready') return readerFailed ? 'reader' : undefined;
  return status;
}

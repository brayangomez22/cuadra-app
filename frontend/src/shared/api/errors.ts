// API errors follow the Error schema: {"error": {"code": "...", "message": "..."}}.
// The message is Spanish and safe to show; the code is stable for decisions.

type ApiErrorBody = { error: { code: string; message: string } };

function isApiError(body: unknown): body is ApiErrorBody {
  if (typeof body !== 'object' || body === null || !('error' in body)) return false;
  const { error } = body;
  return (
    typeof error === 'object' &&
    error !== null &&
    'code' in error &&
    typeof error.code === 'string' &&
    'message' in error &&
    typeof error.message === 'string'
  );
}

// errorCode returns the code of an API error body, or undefined if it is not one.
export function errorCode(body: unknown): string | undefined {
  return isApiError(body) ? body.error.code : undefined;
}

// errorMessage returns the message of an API error body, or the fallback.
export function errorMessage(body: unknown, fallback: string): string {
  return isApiError(body) && body.error.message !== '' ? body.error.message : fallback;
}

// The request did not reach the API (no connection, DNS, server down).
export const NETWORK_ERROR_MESSAGE = 'No hay conexión con Cuadra. Revise el internet e intente de nuevo.';

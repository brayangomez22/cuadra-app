// API errors follow the Error schema: {"error": {"code": "...", "message": "..."}}.
// The message is Spanish and safe to show; the code is stable for decisions.
// Some errors add details, such as the product_id of insufficient_stock.

type ApiErrorBody = { error: { code: string; message: string; details?: Record<string, string> } };

function isObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function isApiError(body: unknown): body is ApiErrorBody {
  if (!isObject(body) || !isObject(body.error)) return false;
  return typeof body.error.code === 'string' && typeof body.error.message === 'string';
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

// ApiError is a response of the API with an error status, thrown by unwrap so
// TanStack Query and the forms handle it like any other failure.
export class ApiError extends Error {
  readonly status: number;
  readonly code: string | undefined;
  readonly details: Record<string, string>;

  constructor(status: number, body: unknown) {
    super(errorMessage(body, 'Algo salió mal en Cuadra. Intente de nuevo.'));
    this.name = 'ApiError';
    this.status = status;
    this.code = errorCode(body);
    this.details = (isApiError(body) ? body.error.details : undefined) ?? {};
  }
}

/** The data of an openapi-fetch result, or an ApiError with its code and message. */
export function unwrap<T>(result: { data?: T; error?: unknown; response: Response }): T {
  if (result.data !== undefined) return result.data;
  throw new ApiError(result.response.status, result.error);
}

/**
 * What to show for a failed request: the API's code and Spanish message, or
 * the no-connection message when the request did not reach the API.
 */
export function describeError(error: unknown): { code?: string; message: string } {
  if (error instanceof ApiError) return { code: error.code, message: error.message };
  return { message: NETWORK_ERROR_MESSAGE };
}

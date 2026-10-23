import type { components } from '@/shared/api/schema';

/** The body a request of a test sent, typed as the contract's. */
export async function sentBody<T>(request: Request): Promise<T> {
  return (await request.json()) as T;
}

/** The ProductInput a request to POST or PUT /products sent. */
export const sentProduct = (request: Request) => sentBody<components['schemas']['ProductInput']>(request);

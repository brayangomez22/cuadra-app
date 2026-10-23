import type { paths } from '@/shared/api/schema';

// The keys of fakeApi, derived from the client generated from api/openapi.yaml:
// a mock of a route or a body the contract no longer has fails in tsc.

type Method = 'get' | 'put' | 'post' | 'delete' | 'patch';

/** "METHOD /path" of every operation in the contract, e.g. "GET /api/v1/products/{productId}". */
export type RouteKey = {
  [P in keyof paths]: {
    [M in Method]: [NonNullable<paths[P][M]>] extends [never] ? never : `${Uppercase<M>} ${P}`;
  }[Method];
}[keyof paths];

type Segments<S extends string> = S extends `${infer Head}/${infer Tail}` ? [Head, ...Segments<Tail>] : [S];

// SameShape compares a concrete path with a template segment by segment: a
// {param} takes one non-empty segment, so products/{id} is not products/x/kardex.
type SameShape<A extends string[], T extends string[]> = A extends [infer a, ...infer ar extends string[]]
  ? T extends [infer t, ...infer tr extends string[]]
    ? t extends `{${string}}`
      ? a extends ''
        ? false
        : SameShape<ar, tr>
      : a extends t
        ? SameShape<ar, tr>
        : false
    : false
  : T extends []
    ? true
    : false;

/** The contract's key a concrete key answers: itself if it is one, or the templates it fits. */
export type ResolveKey<K extends string> = K extends RouteKey
  ? K
  : { [T in RouteKey]: SameShape<Segments<K>, Segments<T>> extends true ? T : never }[RouteKey];

type Operation<K extends string> = K extends `${infer M} ${infer P extends keyof paths}`
  ? NonNullable<paths[P][Lowercase<M> & keyof paths[P]]>
  : never;

type JsonContent<R> = R extends { content: { 'application/json': infer B } } ? B : never;

/** The JSON bodies an operation answers, in any status (the errors included). */
export type ResponseBody<K extends RouteKey> =
  Operation<K> extends { responses: infer Rs } ? JsonContent<Rs[keyof Rs]> : never;

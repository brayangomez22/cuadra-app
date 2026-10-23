import { useCallback } from 'react';
import { useSearchParams } from 'react-router';

/** How a value travels in the URL. `parse` returns undefined for a value it does not accept. */
export type SearchParamCodec<T> = {
  parse: (raw: string) => T | undefined;
  format: (value: T) => string;
  fallback: T;
};

/** A page number: a whole number from 1, which is also the default. */
export const pageParam: SearchParamCodec<number> = {
  parse: (raw) => (/^\d+$/.test(raw) && Number(raw) >= 1 ? Number(raw) : undefined),
  format: String,
  fallback: 1,
};

/** Free text, such as a search. */
export function stringParam(fallback = ''): SearchParamCodec<string> {
  return { parse: (raw) => raw, format: (value) => value, fallback };
}

/** One of a fixed set of values, such as a status filter. */
export function enumParam<T extends string>(values: readonly T[], fallback: T): SearchParamCodec<T> {
  return {
    parse: (raw) => values.find((value) => value === raw),
    format: (value) => value,
    fallback,
  };
}

type Options = {
  /** Parameters removed whenever the value changes, such as the page when a filter changes. */
  reset?: string[];
  /** Replace the history entry instead of adding one (typing a search). */
  replace?: boolean;
};

/**
 * Typed state kept in a URL search parameter, so a link or a reload shows the
 * same page, filter or location. The default value leaves no parameter.
 */
export function useSearchParamsState<T>(key: string, codec: SearchParamCodec<T>, options: Options = {}) {
  const [params, setParams] = useSearchParams();
  const raw = params.get(key);
  const value = (raw === null ? undefined : codec.parse(raw)) ?? codec.fallback;
  const { reset, replace } = options;

  const setValue = useCallback(
    (next: T) => {
      setParams(
        (current) => {
          const updated = new URLSearchParams(current);
          for (const name of reset ?? []) updated.delete(name);
          const formatted = codec.format(next);
          if (formatted === codec.format(codec.fallback)) updated.delete(key);
          else updated.set(key, formatted);
          return updated;
        },
        { replace },
      );
    },
    [codec, key, replace, reset, setParams],
  );

  return [value, setValue] as const;
}

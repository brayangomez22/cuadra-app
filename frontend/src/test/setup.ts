import '@testing-library/jest-dom/vitest';
import { configure } from '@testing-library/react';

// The screen tests render the real routes: a page may chain several requests
// (session, locations, stock, products) and a 300 ms search debounce. With the
// whole suite running in parallel, one second is not always enough.
configure({ asyncUtilTimeout: 3000 });

// jsdom lacks APIs that Radix Select and its popper use in the browser. The
// stylelint and styles-check tests run in Node, without a DOM.
// The types say they always exist, so the check is by name.
function polyfill(target: object, key: string, value: unknown) {
  if (!(key in target)) Object.defineProperty(target, key, { value, configurable: true, writable: true });
}

if (typeof Element !== 'undefined') {
  polyfill(Element.prototype, 'hasPointerCapture', () => false);
  polyfill(Element.prototype, 'releasePointerCapture', () => undefined);
  polyfill(Element.prototype, 'scrollIntoView', () => undefined);
  polyfill(
    globalThis,
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  );
}

// fakeApi stubs the global fetch; each test starts with the real one.
afterEach(() => {
  vi.unstubAllGlobals();
});

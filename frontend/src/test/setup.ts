import '@testing-library/jest-dom/vitest';

// jsdom lacks APIs that Radix Select and its popper use in the browser. The
// stylelint and styles-check tests run in Node, without a DOM.
if (typeof Element !== 'undefined') {
  Element.prototype.hasPointerCapture ??= () => false;
  Element.prototype.releasePointerCapture ??= () => {};
  Element.prototype.scrollIntoView ??= () => {};
  globalThis.ResizeObserver ??= class {
    observe() {}
    unobserve() {}
    disconnect() {}
  };
}

/// <reference types="vite/client" />

declare module '*.module.scss' {
  const classes: Readonly<Record<string, string>>;
  export default classes;
}

interface ImportMetaEnv {
  /** Sentry DSN of the frontend project; empty disables error reporting. */
  readonly VITE_SENTRY_DSN?: string;
  /** development | staging | production; defaults to Vite's mode. */
  readonly VITE_APP_ENV?: string;
  /** Build name in Sentry (the git SHA in the CI); must match the uploaded source maps. */
  readonly VITE_RELEASE?: string;
}

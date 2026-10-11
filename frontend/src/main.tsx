import * as Sentry from '@sentry/react';
import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { App } from '@/app/App';
import { CrashPage } from '@/app/ErrorPage';
import { initSentry } from '@/shared/observability';
import '@/styles/main.scss';

initSentry();

const root = document.getElementById('root');
if (!root) throw new Error('Missing #root element in index.html');

// The router reports the errors of its screens (RouteError). The boundary covers
// the providers around it; the root callbacks, what no boundary caught.
createRoot(root, {
  onUncaughtError: Sentry.reactErrorHandler(),
  onRecoverableError: Sentry.reactErrorHandler(),
}).render(
  <StrictMode>
    <Sentry.ErrorBoundary fallback={<CrashPage />}>
      <App />
    </Sentry.ErrorBoundary>
  </StrictMode>,
);

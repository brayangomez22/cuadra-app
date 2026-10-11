import * as Sentry from '@sentry/react';
import { useEffect } from 'react';
import { isRouteErrorResponse, useRouteError } from 'react-router';
import { CrashPage, NotFoundPage } from './ErrorPage';

// RouteError is the root route's errorElement. The router catches what a screen
// throws while rendering or loading before any outer error boundary sees it, so
// this is where those errors are reported. A URL that matches no route also ends
// here; that is not a bug and is not reported.
export function RouteError() {
  const error = useRouteError();
  const notFound = isRouteErrorResponse(error) && error.status === 404;

  useEffect(() => {
    if (!notFound) Sentry.captureException(error);
  }, [error, notFound]);

  return notFound ? <NotFoundPage /> : <CrashPage />;
}

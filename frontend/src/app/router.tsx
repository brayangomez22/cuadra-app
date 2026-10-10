import { createBrowserRouter } from 'react-router';

// The screens arrive in T15 (auth and layout) and T16 (catalog and inventory).
export const router = createBrowserRouter([
  {
    path: '/',
    element: <h1>Cuadra</h1>,
  },
]);

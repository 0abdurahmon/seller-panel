import { RouterProvider } from 'react-router/dom';
import { SkulletProvider } from 'skullet/react';
import { QueryProvider } from '@/shared/providers';
import { router } from './router';

export const App = () => {
  return (
    <QueryProvider>
      <SkulletProvider reveal="sweep">
        <RouterProvider router={router} />
      </SkulletProvider>
    </QueryProvider>
  );
};

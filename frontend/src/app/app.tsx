import { RouterProvider } from 'react-router/dom';
import { sweep } from 'skullet';
import { SkulletProvider } from 'skullet/react';
import { QueryProvider } from '@/shared/providers';
import { router } from './router';

// Custom sweep with a zero hold time so the skeleton always renders immediately,
// giving the minimum API delay (450ms) full headroom to display the shimmer & sweep.
const sweepReveal = {
  ...sweep,
  hold: 0,
};

export const App = () => {
  return (
    <QueryProvider>
      <SkulletProvider reveal={sweepReveal}>
        <RouterProvider router={router} />
      </SkulletProvider>
    </QueryProvider>
  );
};

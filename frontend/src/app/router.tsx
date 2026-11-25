import { lazy, Suspense, type ComponentType } from 'react';
import { createBrowserRouter } from 'react-router';
import type { RouteHandle } from '@/widgets/header';
import { AuthLayout } from './auth-layout';
import { AppLayout } from './layout';
import { GuestOnly, RequireSession } from './route-guards';

// `handle.title` is the page title the header shows.
const title = (value: string): RouteHandle => ({ title: value });

/**
 * Each page is its own chunk. React.lazy (not the route `lazy` option) keeps the layout on
 * screen while the chunk loads; the page then shows its own loading state (skeletons).
 */
function page(load: () => Promise<ComponentType>) {
  const Page = lazy(() => load().then((Component) => ({ default: Component })));
  return (
    <Suspense fallback={null}>
      <Page />
    </Suspense>
  );
}

export const router = createBrowserRouter([
  {
    element: <GuestOnly />,
    children: [
      {
        element: <AuthLayout />,
        children: [
          {
            path: '/login',
            element: page(() => import('@/pages/login').then((m) => m.LoginPage)),
          },
          {
            path: '/login/otp',
            element: page(() => import('@/pages/otp').then((m) => m.OtpPage)),
          },
          {
            path: '/register',
            element: page(() => import('@/pages/register').then((m) => m.RegisterPage)),
          },
        ],
      },
    ],
  },
  {
    element: <RequireSession />,
    children: [
      {
        element: <AppLayout />,
        children: [
          {
            path: '/',
            handle: title('Дашборд'),
            element: page(() => import('@/pages/dashboard').then((m) => m.DashboardPage)),
          },
          {
            path: '/products',
            handle: title('Товары'),
            element: page(() => import('@/pages/products').then((m) => m.ProductsPage)),
          },
          {
            path: '/orders',
            handle: title('Заказы'),
            element: page(() => import('@/pages/orders').then((m) => m.OrdersPage)),
          },
          {
            path: '/balance',
            handle: title('Баланс'),
            element: page(() => import('@/pages/balance').then((m) => m.BalancePage)),
          },
          {
            path: '/shop',
            handle: title('Магазин'),
            element: page(() => import('@/pages/shop').then((m) => m.ShopPage)),
          },
          {
            path: '*',
            handle: title('Страница не найдена'),
            element: page(() => import('@/pages/not-found').then((m) => m.NotFoundPage)),
          },
        ],
      },
    ],
  },
]);

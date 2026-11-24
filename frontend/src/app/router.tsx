import { createBrowserRouter } from 'react-router';
import { Spinner } from '@/shared/ui';
import type { RouteHandle } from '@/widgets/header';
import { AppLayout } from './layout';

// `handle.title` is the page title the header shows.
const title = (value: string): RouteHandle => ({ title: value });

// Each page is its own chunk, so the first load only fetches the page that is opened.
export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    // Shown only on the first load, while the opened page's chunk arrives.
    HydrateFallback: () => (
      <div className="flex h-dvh items-center justify-center">
        <Spinner className="size-6 text-muted-foreground" />
      </div>
    ),
    children: [
      {
        path: '/',
        handle: title('Дашборд'),
        lazy: () => import('@/pages/dashboard').then((m) => ({ Component: m.DashboardPage })),
      },
      {
        path: '/products',
        handle: title('Товары'),
        lazy: () => import('@/pages/products').then((m) => ({ Component: m.ProductsPage })),
      },
      {
        path: '/orders',
        handle: title('Заказы'),
        lazy: () => import('@/pages/orders').then((m) => ({ Component: m.OrdersPage })),
      },
      {
        path: '/balance',
        handle: title('Баланс'),
        lazy: () => import('@/pages/balance').then((m) => ({ Component: m.BalancePage })),
      },
      {
        path: '/shop',
        handle: title('Магазин'),
        lazy: () => import('@/pages/shop').then((m) => ({ Component: m.ShopPage })),
      },
      {
        path: '*',
        handle: title('Страница не найдена'),
        lazy: () => import('@/pages/not-found').then((m) => ({ Component: m.NotFoundPage })),
      },
    ],
  },
]);

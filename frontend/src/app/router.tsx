import { createBrowserRouter } from 'react-router';
import { BalancePage } from '@/pages/balance';
import { DashboardPage } from '@/pages/dashboard';
import { NotFoundPage } from '@/pages/not-found';
import { OrdersPage } from '@/pages/orders';
import { ProductsPage } from '@/pages/products';
import { ShopPage } from '@/pages/shop';
import type { RouteHandle } from '@/widgets/header';
import { AppLayout } from './layout';

// `handle.title` is the page title the header shows.
const title = (value: string): RouteHandle => ({ title: value });

export const router = createBrowserRouter([
  {
    element: <AppLayout />,
    children: [
      { path: '/', element: <DashboardPage />, handle: title('Дашборд') },
      { path: '/products', element: <ProductsPage />, handle: title('Товары') },
      { path: '/orders', element: <OrdersPage />, handle: title('Заказы') },
      { path: '/balance', element: <BalancePage />, handle: title('Баланс') },
      { path: '/shop', element: <ShopPage />, handle: title('Магазин') },
      { path: '*', element: <NotFoundPage />, handle: title('Страница не найдена') },
    ],
  },
]);

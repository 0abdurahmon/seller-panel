import type { DashboardSummary } from '@/shared/api';

// Temporary data in the shape of GET /dashboard/summary until the screen is wired to the API.
// Delete this file then.
export const MOCK_SUMMARY: DashboardSummary = {
  today_orders: 12,
  waiting_for_action: 5,
  revenue_7d: 18_450_000,
  revenue_30d: 64_200_000,
  top_products: [
    { product_id: 1, title: 'Смартфон Galaxy A55 8/256', quantity_sold: 42, revenue: 20_538_000 },
    { product_id: 2, title: 'Наушники беспроводные Pro', quantity_sold: 31, revenue: 3_875_000 },
    { product_id: 3, title: 'Зарядное устройство 65W USB-C', quantity_sold: 27, revenue: 8_640_000 },
    { product_id: 4, title: 'Смарт-часы Fit 3', quantity_sold: 18, revenue: 12_420_000 },
    { product_id: 5, title: 'Чехол силиконовый, чёрный', quantity_sold: 12, revenue: 1_068_000 },
  ],
};

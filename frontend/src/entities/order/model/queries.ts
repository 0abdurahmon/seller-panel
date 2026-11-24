import { keepPreviousData, useQueries, useQuery } from '@tanstack/react-query';
import { getOrders } from '@/shared/api';
import type { GetOrdersParams } from '@/shared/api';
import type { OrderStatusFilter } from './types';

const api = getOrders();

export const orderKeys = {
  all: ['orders'] as const,
  list: (params: GetOrdersParams) => [...orderKeys.all, 'list', params] as const,
  detail: (id: number) => [...orderKeys.all, 'detail', id] as const,
  count: (status: OrderStatusFilter) => [...orderKeys.all, 'count', status] as const,
};

export function useOrders(params: GetOrdersParams) {
  return useQuery({
    queryKey: orderKeys.list(params),
    queryFn: ({ signal }) => api.getOrders(params, { signal }),
    // Keeps the current page on screen while the next one loads.
    placeholderData: keepPreviousData,
  });
}

/** One order with its items and status history; nothing is requested while `id` is null. */
export function useOrder(id: number | null) {
  return useQuery({
    queryKey: orderKeys.detail(id ?? 0),
    queryFn: ({ signal }) => api.getOrdersId(id as number, { signal }),
    enabled: id !== null,
  });
}

const COUNTED: OrderStatusFilter[] = [
  'all',
  'new',
  'accepted',
  'packed',
  'handed_to_courier',
  'delivered',
  'cancelled',
];

/** Totals for the status tabs: one request per tab with page_size=1, only `total` is used. */
export function useOrderCounts(): Partial<Record<OrderStatusFilter, number>> {
  return useQueries({
    queries: COUNTED.map((status) => ({
      queryKey: orderKeys.count(status),
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        api.getOrders({ page_size: 1, ...(status === 'all' ? {} : { status }) }, { signal }),
      placeholderData: keepPreviousData,
    })),
    combine: (results) =>
      Object.fromEntries(COUNTED.map((status, i) => [status, results[i].data?.total])),
  });
}

import { keepPreviousData, useInfiniteQuery, useQueries, useQuery } from '@tanstack/react-query';
import { getOrders } from '@/shared/api';
import type { GetOrdersParams, OrderStatus } from '@/shared/api';
import type { OrderStatusFilter } from './types';

const api = getOrders();

const WAITING_STATUSES: OrderStatus[] = ['new', 'accepted', 'packed'];

export const orderKeys = {
  all: ['orders'] as const,
  list: (params: Omit<GetOrdersParams, 'page' | 'status'> & { status?: OrderStatusFilter }) =>
    [...orderKeys.all, 'list', params] as const,
  detail: (id: number) => [...orderKeys.all, 'detail', id] as const,
  count: (status: Exclude<OrderStatusFilter, 'waiting'>) =>
    [...orderKeys.all, 'count', status] as const,
};

/** Pages of orders for infinite scroll; `params` are the filters, the page number is managed here. */
export function useOrders(
  params: Omit<GetOrdersParams, 'page' | 'status'> & { status?: OrderStatusFilter },
) {
  const { status, ...rest } = params;

  return useInfiniteQuery({
    queryKey: orderKeys.list(params),
    queryFn: async ({ pageParam, signal }) => {
      if (status === 'waiting') {
        const pages = await Promise.all(
          WAITING_STATUSES.map((s) =>
            api.getOrders({ ...rest, page: pageParam, status: s }, { signal }),
          ),
        );
        const items = pages
          .flatMap((p) => p.items)
          .sort(
            (a, b) =>
              new Date(b.created_at).getTime() - new Date(a.created_at).getTime() || b.id - a.id,
          );
        const hasMore = pages.some((p) => p.page * p.page_size < p.total);
        const total = pages.reduce((sum, p) => sum + p.total, 0);
        return {
          items,
          page: pageParam,
          page_size: rest.page_size ?? 20,
          total,
          hasMore,
        };
      }

      const res = await api.getOrders(
        { ...rest, page: pageParam, ...(status && status !== 'all' ? { status } : {}) },
        { signal },
      );
      return {
        ...res,
        hasMore: res.page * res.page_size < res.total,
      };
    },
    initialPageParam: 1,
    getNextPageParam: (last) => (last.hasMore ? last.page + 1 : undefined),
    select: (data) => ({
      items: data.pages.flatMap((page) => page.items),
      total: data.pages[0]?.total ?? 0,
    }),
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

const COUNTED: Exclude<OrderStatusFilter, 'waiting'>[] = [
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
    combine: (results) => {
      const counts: Partial<Record<OrderStatusFilter, number>> = Object.fromEntries(
        COUNTED.map((status, i) => [status, results[i].data?.total]),
      );
      const waitingLoaded = WAITING_STATUSES.every((s) => counts[s] !== undefined);
      if (waitingLoaded) {
        counts.waiting = WAITING_STATUSES.reduce((sum, s) => sum + (counts[s] ?? 0), 0);
      }
      return counts;
    },
  });
}

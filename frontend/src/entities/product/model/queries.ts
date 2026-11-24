import { keepPreviousData, useQueries, useQuery } from '@tanstack/react-query';
import { getProducts } from '@/shared/api/generated';
import type { GetProductsParams } from '@/shared/api/model';
import type { ProductStatusFilter } from './types';

const api = getProducts();

export const productKeys = {
  all: ['products'] as const,
  list: (params: GetProductsParams) => [...productKeys.all, 'list', params] as const,
  count: (status: ProductStatusFilter) => [...productKeys.all, 'count', status] as const,
};

export function useProducts(params: GetProductsParams) {
  return useQuery({
    queryKey: productKeys.list(params),
    queryFn: ({ signal }) => api.getProducts(params, { signal }),
    // Keeps the current page on screen while the next one loads.
    placeholderData: keepPreviousData,
  });
}

const COUNTED: ProductStatusFilter[] = ['all', 'active', 'draft', 'archived'];

/** Totals for the status tabs: one request per tab with page_size=1, only `total` is used. */
export function useProductCounts(): Partial<Record<ProductStatusFilter, number>> {
  return useQueries({
    queries: COUNTED.map((status) => ({
      queryKey: productKeys.count(status),
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        api.getProducts({ page_size: 1, ...(status === 'all' ? {} : { status }) }, { signal }),
    })),
    combine: (results) =>
      Object.fromEntries(COUNTED.map((status, i) => [status, results[i].data?.total])),
  });
}

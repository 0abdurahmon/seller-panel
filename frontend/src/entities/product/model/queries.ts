import { keepPreviousData, useQueries, useQuery } from '@tanstack/react-query';
import { getProducts } from '@/shared/api';
import type { GetProductsParams } from '@/shared/api';
import type { ProductStatusFilter } from './types';

const api = getProducts();

export const productKeys = {
  all: ['products'] as const,
  list: (params: GetProductsParams) => [...productKeys.all, 'list', params] as const,
  count: (status: ProductStatusFilter, filters: ProductSearchFilters) =>
    [...productKeys.all, 'count', status, filters] as const,
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

/** Search and category, the filters the status tabs count within. */
export type ProductSearchFilters = Pick<GetProductsParams, 'q' | 'category'>;

/**
 * Totals for the status tabs, within the current search and category:
 * one request per tab with page_size=1, only `total` is used.
 */
export function useProductCounts(
  filters: ProductSearchFilters = {},
): Partial<Record<ProductStatusFilter, number>> {
  return useQueries({
    queries: COUNTED.map((status) => ({
      queryKey: productKeys.count(status, filters),
      queryFn: ({ signal }: { signal: AbortSignal }) =>
        api.getProducts(
          { ...filters, page_size: 1, ...(status === 'all' ? {} : { status }) },
          { signal },
        ),
      placeholderData: keepPreviousData,
    })),
    combine: (results) =>
      Object.fromEntries(COUNTED.map((status, i) => [status, results[i].data?.total])),
  });
}

import { queryOptions, useInfiniteQuery, useQuery } from '@tanstack/react-query';
import { getBalance, type GetBalanceEntriesParams } from '@/shared/api';

const api = getBalance();

export const balanceKeys = {
  all: ['balance'] as const,
  summary: () => [...balanceKeys.all, 'summary'] as const,
  entries: () => [...balanceKeys.all, 'entries'] as const,
  entriesList: (params?: Omit<GetBalanceEntriesParams, 'page'>) =>
    [...balanceKeys.entries(), params ?? {}] as const,
};

export const balanceQueries = {
  summary: () =>
    queryOptions({
      queryKey: balanceKeys.summary(),
      queryFn: ({ signal }) => api.getBalance({ signal }),
    }),
};

export function useBalance() {
  return useQuery(balanceQueries.summary());
}

/** Pages of balance entries for infinite scroll; page number is handled internally. */
export function useBalanceEntries(params?: Omit<GetBalanceEntriesParams, 'page'>) {
  return useInfiniteQuery({
    queryKey: balanceKeys.entriesList(params),
    queryFn: ({ pageParam, signal }) =>
      api.getBalanceEntries({ ...params, page: pageParam }, { signal }),
    initialPageParam: 1,
    getNextPageParam: (last) =>
      last.page * last.page_size < last.total ? last.page + 1 : undefined,
    select: (data) => ({
      items: data.pages.flatMap((page) => page.items),
      total: data.pages[0]?.total ?? 0,
    }),
  });
}

import { useEffect, useRef } from 'react';
import { useBalance, useBalanceEntries } from '@/entities/balance';
import { getApiErrorMessage } from '@/shared/api';
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
  Button,
} from '@/shared/ui';
import { BalanceSummary } from '@/widgets/balance-summary';
import { BalanceTable } from '@/widgets/balance-table';

function getScrollParent(element: HTMLElement): HTMLElement | null {
  let parent = element.parentElement;
  while (parent) {
    const { overflowY } = getComputedStyle(parent);
    if (overflowY === 'auto' || overflowY === 'scroll') return parent;
    parent = parent.parentElement;
  }
  return null;
}

const PAGE_SIZE = 20;

export const BalancePage = () => {
  const balanceQuery = useBalance();
  const entriesQuery = useBalanceEntries({ page_size: PAGE_SIZE });

  const sentinel = useRef<HTMLDivElement>(null);
  const { hasNextPage, isFetchingNextPage, isFetchNextPageError, fetchNextPage } = entriesQuery;

  useEffect(() => {
    const node = sentinel.current;
    if (!node || !hasNextPage || isFetchingNextPage || isFetchNextPageError) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) void fetchNextPage();
      },
      { root: getScrollParent(node), rootMargin: '0px 0px 800px 0px' }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, isFetchNextPageError, fetchNextPage]);

  if (balanceQuery.isError && !balanceQuery.data && entriesQuery.isError && !entriesQuery.data) {
    const error = balanceQuery.error ?? entriesQuery.error;
    return (
      <Alert variant="destructive">
        <AlertTitle>Не удалось загрузить баланс</AlertTitle>
        <AlertDescription>{getApiErrorMessage(error)}</AlertDescription>
        <AlertAction>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              balanceQuery.refetch();
              entriesQuery.refetch();
            }}
          >
            Попробовать снова
          </Button>
        </AlertAction>
      </Alert>
    );
  }

  return (
    <div className="flex flex-1 flex-col gap-6">
      <BalanceSummary
        balance={balanceQuery.data}
        isLoading={balanceQuery.isLoading}
      />

      <div className="flex flex-1 flex-col gap-3">
        <h2 className="text-lg font-semibold tracking-tight text-foreground">
          История операций
        </h2>

        <BalanceTable
          entries={entriesQuery.data?.items}
          total={entriesQuery.data?.total}
          isLoading={entriesQuery.isLoading}
          hasNextPage={hasNextPage}
          isFetchingNextPage={isFetchingNextPage}
          isFetchNextPageError={isFetchNextPageError}
          onRetryNextPage={() => void fetchNextPage()}
          sentinelRef={sentinel}
        />
      </div>
    </div>
  );
};

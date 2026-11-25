import { Link } from 'react-router';
import { Wallet } from 'iconsax-reactjs';
import { Skeleton } from 'skullet/react';
import { EntryTypeBadge, type BalanceEntry } from '@/entities/balance';
import { formatDateTime, formatPrice } from '@/shared/lib';
import {
  Button,
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
  Spinner,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/shared/ui';

// Stands in for real entries while loading, so skullet can measure the table's shape.
const SAMPLE_ENTRIES: BalanceEntry[] = [
  {
    id: -1,
    type: 'sale',
    amount: 1_250_000,
    order_id: 1042,
    created_at: '2026-11-25T08:30:00Z',
  },
  {
    id: -2,
    type: 'commission',
    amount: -125_000,
    order_id: 1042,
    created_at: '2026-11-25T08:30:00Z',
  },
  {
    id: -3,
    type: 'sale',
    amount: 3_400_000,
    order_id: 1041,
    created_at: '2026-11-24T16:20:00Z',
  },
  {
    id: -4,
    type: 'commission',
    amount: -340_000,
    order_id: 1041,
    created_at: '2026-11-24T16:20:00Z',
  },
  {
    id: -5,
    type: 'payout',
    amount: -10_000_000,
    order_id: null,
    created_at: '2026-11-20T11:00:00Z',
  },
];

function BalanceEmptyState() {
  return (
    <Empty className="flex-1 rounded-3xl border border-dashed border-border bg-card">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Wallet className="text-primary" />
        </EmptyMedia>
        <EmptyTitle>Операций пока нет</EmptyTitle>
        <EmptyDescription>
          Начисления появятся здесь, когда ваши первые заказы будут доставлены покупателям.
        </EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
}

function BalanceTableView({ entries }: { entries: BalanceEntry[] }) {
  const formatAmount = (entry: BalanceEntry) => {
    if (entry.type === 'sale') {
      return (
        <span className="font-medium text-success tabular-nums">
          +{formatPrice(entry.amount)}
        </span>
      );
    }
    if (entry.type === 'commission') {
      return (
        <span className="font-medium text-muted-foreground tabular-nums">
          -{formatPrice(Math.abs(entry.amount))}
        </span>
      );
    }
    return (
      <span className="font-medium text-destructive tabular-nums">
        -{formatPrice(Math.abs(entry.amount))}
      </span>
    );
  };

  return (
    <div className="overflow-hidden rounded-3xl border border-border bg-card">
      <Table>
        <TableHeader>
          <TableRow>
            <TableHead className="w-[180px]">Дата и время</TableHead>
            <TableHead className="w-[140px]">Операция</TableHead>
            <TableHead>Основание</TableHead>
            <TableHead className="text-right">Сумма</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {entries.map((entry) => (
            <TableRow key={entry.id}>
              <TableCell className="text-xs text-muted-foreground tabular-nums">
                {formatDateTime(entry.created_at)}
              </TableCell>
              <TableCell>
                <EntryTypeBadge type={entry.type} />
              </TableCell>
              <TableCell>
                {entry.order_id ? (
                  <Button
                    variant="link"
                    className="h-auto p-0 font-medium"
                    nativeButton={false}
                    render={<Link to={`/orders?order=${entry.order_id}`} />}
                  >
                    Заказ #{entry.order_id}
                  </Button>
                ) : (
                  <span className="text-muted-foreground">—</span>
                )}
              </TableCell>
              <TableCell className="text-right">{formatAmount(entry)}</TableCell>
            </TableRow>
          ))}
        </TableBody>
      </Table>
    </div>
  );
}

interface BalanceTableProps {
  entries: BalanceEntry[] | undefined;
  total?: number;
  isLoading?: boolean;
  hasNextPage?: boolean;
  isFetchingNextPage?: boolean;
  isFetchNextPageError?: boolean;
  onRetryNextPage?: () => void;
  sentinelRef?: React.RefObject<HTMLDivElement | null>;
}

export const BalanceTable = ({
  entries,
  total = 0,
  isLoading,
  hasNextPage,
  isFetchingNextPage,
  isFetchNextPageError,
  onRetryNextPage,
  sentinelRef,
}: BalanceTableProps) => {
  const isEmpty = !isLoading && entries?.length === 0;

  return (
    <div className="flex flex-1 flex-col gap-4">
      <Skeleton
        id="balance-table"
        className={isEmpty ? "flex flex-1 flex-col *:flex *:flex-1 *:flex-col" : undefined}
        label="Загрузка истории операций"
        data={isLoading ? null : entries}
        sample={SAMPLE_ENTRIES}
        empty={isEmpty}
        emptyMessage={<BalanceEmptyState />}
      >
        {(rows) => <BalanceTableView entries={rows} />}
      </Skeleton>

      {!isEmpty && !isLoading && entries && (
        <>
          <div className="flex items-center justify-between text-xs text-muted-foreground">
            <span>
              Показано: {entries.length} из {total}
            </span>
            {hasNextPage && !isFetchingNextPage && (
              <span>Прокрутите вниз для загрузки</span>
            )}
          </div>

          {isFetchNextPageError ? (
            <div className="flex items-center justify-center gap-3 py-2 text-sm text-muted-foreground">
              <span>Не удалось загрузить следующие операции.</span>
              <Button size="sm" variant="outline" onClick={onRetryNextPage}>
                Повторить
              </Button>
            </div>
          ) : (
            <div ref={sentinelRef} className="py-2 text-center text-sm text-muted-foreground">
              {isFetchingNextPage && (
                <div className="flex items-center justify-center gap-2">
                  <Spinner />
                  <span>Загружаем ещё…</span>
                </div>
              )}
            </div>
          )}
        </>
      )}
    </div>
  );
};

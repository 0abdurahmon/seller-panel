import { ReceiptText, TrendUp, Wallet } from 'iconsax-reactjs';
import { Skeleton } from 'skullet/react';
import type { Balance } from '@/entities/balance';
import { AnimatedNumber, Card } from '@/shared/ui';

// Stands in for the real numbers while loading, so skullet can measure the layout.
const SAMPLE_BALANCE: Balance = {
  available: 24_500_000,
  earned: 27_220_000,
  commission: -2_722_000,
};

function BalanceSummaryView({ balance }: { balance: Balance }) {
  return (
    <div className="grid gap-4 sm:grid-cols-3">
      <Card className="gap-3 px-5 py-5 shadow-none">
        <div className="flex items-center justify-between gap-3">
          <span className="text-sm font-medium text-muted-foreground">Доступно к выводу</span>
          <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground [&_svg]:size-[18px]">
            <Wallet />
          </span>
        </div>
        <span className="text-3xl font-semibold tracking-tight whitespace-nowrap text-foreground">
          <AnimatedNumber value={balance.available} />
          <span className="ml-1.5 text-base font-medium text-muted-foreground">сум</span>
        </span>
        <span className="text-xs text-muted-foreground">
          Фактический остаток на счете магазина
        </span>
      </Card>

      <Card className="gap-3 px-5 py-5 shadow-none">
        <div className="flex items-center justify-between gap-3">
          <span className="text-sm font-medium text-muted-foreground">Всего заработано</span>
          <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground [&_svg]:size-[18px]">
            <TrendUp />
          </span>
        </div>
        <span className="text-3xl font-semibold tracking-tight whitespace-nowrap text-foreground">
          <AnimatedNumber value={balance.earned} />
          <span className="ml-1.5 text-base font-medium text-muted-foreground">сум</span>
        </span>
        <span className="text-xs text-muted-foreground">
          Сумма доставленных заказов без вычета комиссии
        </span>
      </Card>

      <Card className="gap-3 px-5 py-5 shadow-none">
        <div className="flex items-center justify-between gap-3">
          <span className="text-sm font-medium text-muted-foreground">Комиссия сервиса</span>
          <span className="flex size-9 shrink-0 items-center justify-center rounded-xl bg-muted text-muted-foreground [&_svg]:size-[18px]">
            <ReceiptText />
          </span>
        </div>
        <span className="text-3xl font-semibold tracking-tight whitespace-nowrap text-foreground">
          <AnimatedNumber value={Math.abs(balance.commission)} />
          <span className="ml-1.5 text-base font-medium text-muted-foreground">сум</span>
        </span>
        <span className="text-xs text-muted-foreground">
          10% удержано платформой с доставленных заказов
        </span>
      </Card>
    </div>
  );
}

export const BalanceSummary = ({
  balance,
  isLoading,
}: {
  balance?: Balance;
  isLoading?: boolean;
}) => {
  return (
    <Skeleton
      id="balance-summary"
      label="Загрузка сводки баланса"
      data={isLoading ? null : balance}
      sample={SAMPLE_BALANCE}
    >
      {(data) => <BalanceSummaryView balance={data} />}
    </Skeleton>
  );
};

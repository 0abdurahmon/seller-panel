import { Calendar, Chart, Clock, TrendUp } from 'iconsax-reactjs';
import type { DashboardSummary } from '@/shared/api';
import { StatCard } from './stat-card';

/** The four numbers of MVP item 6: today, waiting for action, revenue for 7 and 30 days. */
export const DashboardStats = ({ summary }: { summary: DashboardSummary }) => {
  const waiting = summary.waiting_for_action;

  return (
    <section aria-label="Сводка" className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
      <StatCard
        label="Заказы сегодня"
        value={summary.today_orders}
        hint="Созданы сегодня, в любом статусе"
        icon={<Calendar />}
      />
      <StatCard
        label="Ждут действия"
        value={waiting}
        hint={waiting > 0 ? 'Новые, принятые и собранные' : 'Всё обработано'}
        icon={<Clock />}
        to={waiting > 0 ? '/orders' : undefined}
      />
      <StatCard
        label="Выручка за 7 дней"
        value={summary.revenue_7d}
        unit="сум"
        hint="Сегодня и 6 дней до, доставленные заказы"
        icon={<TrendUp />}
      />
      <StatCard
        label="Выручка за 30 дней"
        value={summary.revenue_30d}
        unit="сум"
        hint="Сегодня и 29 дней до, доставленные заказы"
        icon={<Chart />}
      />
    </section>
  );
};

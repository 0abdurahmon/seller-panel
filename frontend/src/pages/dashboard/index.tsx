import { Link } from "react-router";
import { Skeleton } from "skullet/react";
import { useDashboardSummary } from "@/entities/dashboard";
import { getApiErrorMessage, type DashboardSummary } from "@/shared/api";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
  Button,
} from "@/shared/ui";
import { DashboardStats } from "@/widgets/dashboard-stats";
import { TopProducts } from "@/widgets/top-products";

// Stands in for the real numbers while loading, so skullet can measure the layout.
const SAMPLE: DashboardSummary = {
  today_orders: 10,
  waiting_for_action: 0,
  revenue_7d: 10_000_000,
  revenue_30d: 10_000_000,
  top_products: Array.from({ length: 5 }, (_, i) => ({
    product_id: -1 - i,
    title: "Смартфон Galaxy A55 8/256",
    quantity_sold: 10,
    revenue: 1_000_000,
  })),
};

function DashboardContent({ summary }: { summary: DashboardSummary }) {
  return (
    <div className="flex flex-col gap-6">
      <div className="flex flex-col gap-2">
        <DashboardStats summary={summary} />
        <p className="text-xs text-muted-foreground">
          Выручка до вычета комиссии 10%. Сколько осталось вам, показывает{" "}
          <Button variant="link" className="h-auto p-0 text-xs" nativeButton={false} render={<Link to="/balance" />}>
            Баланс
          </Button>
          .
        </p>
      </div>
      <TopProducts products={summary.top_products} />
    </div>
  );
}

export const DashboardPage = () => {
  const summary = useDashboardSummary();

  if (summary.isError) {
    return (
      <Alert variant="destructive">
        <AlertTitle>Не удалось загрузить сводку</AlertTitle>
        <AlertDescription>{getApiErrorMessage(summary.error)}</AlertDescription>
        <AlertAction>
          <Button size="sm" variant="outline" onClick={() => summary.refetch()}>
            Повторить
          </Button>
        </AlertAction>
      </Alert>
    );
  }

  return (
    <Skeleton
      id="dashboard"
      label="Загрузка сводки"
      data={summary.data ?? null}
      sample={SAMPLE}
    >
      {(data) => <DashboardContent summary={data} />}
    </Skeleton>
  );
};

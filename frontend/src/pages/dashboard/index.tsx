import { Category } from "iconsax-reactjs";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui";

export const DashboardPage = () => {
  return (
    <Empty className="rounded-3xl border border-dashed border-border bg-card">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Category className="text-primary" />
        </EmptyMedia>
        <EmptyTitle>Дашборд скоро появится</EmptyTitle>
        <EmptyDescription>Здесь будут заказы за сегодня, выручка и топ товаров.</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
};

import { ReceiptText } from "iconsax-reactjs";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui";

export const OrdersPage = () => {
  return (
    <Empty className="rounded-3xl border border-dashed border-border bg-card">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <ReceiptText className="text-primary" />
        </EmptyMedia>
        <EmptyTitle>Заказы скоро появятся</EmptyTitle>
        <EmptyDescription>Здесь можно будет принимать заказы и менять их статус.</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
};

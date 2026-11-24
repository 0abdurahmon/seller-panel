import { Wallet } from "iconsax-reactjs";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui";

export const BalancePage = () => {
  return (
    <Empty className="rounded-3xl border border-dashed border-border bg-card">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Wallet className="text-primary" />
        </EmptyMedia>
        <EmptyTitle>Баланс скоро появится</EmptyTitle>
        <EmptyDescription>Здесь будут заработок, комиссия и история начислений.</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
};

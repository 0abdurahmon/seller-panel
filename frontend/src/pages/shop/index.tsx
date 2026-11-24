import { Shop } from "iconsax-reactjs";
import { Empty, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/shared/ui";

export const ShopPage = () => {
  return (
    <Empty className="rounded-3xl border border-dashed border-border bg-card">
      <EmptyHeader>
        <EmptyMedia variant="icon">
          <Shop className="text-primary" />
        </EmptyMedia>
        <EmptyTitle>Профиль магазина скоро появится</EmptyTitle>
        <EmptyDescription>Здесь можно будет изменить название, логотип и контакты магазина.</EmptyDescription>
      </EmptyHeader>
    </Empty>
  );
};

import type { ReactNode } from "react";
import { cn } from "cn";
import { Box, ReceiptText, Wallet } from "iconsax-reactjs";

const POINTS: { icon: ReactNode; title: string; text: string }[] = [
  {
    icon: <Box />,
    title: "Товары и остатки",
    text: "Карточки, цены и склад в одной таблице.",
  },
  {
    icon: <ReceiptText />,
    title: "Заказы",
    text: "Принимайте, собирайте и передавайте курьеру.",
  },
  {
    icon: <Wallet />,
    title: "Баланс",
    text: "Сколько заработано и какая комиссия удержана.",
  },
];

/** The brand side of the sign-in screens: what the panel is for. */
export const AuthPromo = ({ className }: { className?: string }) => {
  return (
    <aside
      className={cn(
        "flex flex-col justify-between gap-10 bg-primary p-10 text-primary-foreground",
        className,
      )}
    >
      <div className="max-w-md space-y-4">
        <h2 className="text-3xl font-semibold tracking-tight text-balance">
          Ваш магазин на маркетплейсе в одном окне
        </h2>
        <p className="text-base text-primary-foreground/80">
          Dona помогает продавцу вести товары, обрабатывать заказы и следить за деньгами.
        </p>
      </div>

      <ul className="space-y-5">
        {POINTS.map((point) => (
          <li key={point.title} className="flex items-start gap-4">
            <span className="flex size-11 shrink-0 items-center justify-center rounded-2xl bg-primary-foreground/15 [&_svg]:size-5">
              {point.icon}
            </span>
            <span className="space-y-0.5">
              <span className="block font-medium">{point.title}</span>
              <span className="block text-sm text-primary-foreground/80">{point.text}</span>
            </span>
          </li>
        ))}
      </ul>
    </aside>
  );
};

import { useState } from "react";
import { Filter } from "iconsax-reactjs";
import type { OrderStatusFilter } from "@/entities/order";
import {
  Button,
  Credenza,
  CredenzaContent,
  CredenzaDescription,
  CredenzaHeader,
  CredenzaTitle,
  Field,
  FieldLabel,
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/shared/ui";

export type OrderPeriodFilter = "all" | "today" | "7d" | "30d";

const STATUS_ITEMS: { value: OrderStatusFilter; label: string }[] = [
  { value: "all", label: "Все" },
  { value: "waiting", label: "Ожидают действия" },
  { value: "new", label: "Новые" },
  { value: "accepted", label: "Принятые" },
  { value: "packed", label: "Собранные" },
  { value: "handed_to_courier", label: "У курьера" },
  { value: "delivered", label: "Доставленные" },
  { value: "cancelled", label: "Отменённые" },
];

const PERIOD_ITEMS: { value: OrderPeriodFilter; label: string }[] = [
  { value: "all", label: "За всё время" },
  { value: "today", label: "Сегодня" },
  { value: "7d", label: "За 7 дней" },
  { value: "30d", label: "За 30 дней" },
];

export type OrderFiltersButtonProps = {
  status: OrderStatusFilter;
  period: OrderPeriodFilter;
  onStatusChange: (next: OrderStatusFilter) => void;
  onPeriodChange: (next: OrderPeriodFilter) => void;
  onReset: () => void;
};

export const OrderFiltersButton = ({
  status,
  period,
  onStatusChange,
  onPeriodChange,
  onReset,
}: OrderFiltersButtonProps) => {
  const [open, setOpen] = useState(false);
  const hasFilters = status !== "all" || period !== "all";

  return (
    <>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              type="button"
              variant="secondary"
              size="icon"
              className="relative [&_svg:not([class*='size-'])]:size-5"
              aria-label="Фильтры заказов"
              onClick={() => setOpen(true)}
            />
          }
        >
          <Filter />
          {hasFilters && (
            <span
              aria-hidden
              className="absolute top-1.5 right-1.5 size-2 rounded-full bg-primary"
            />
          )}
        </TooltipTrigger>
        <TooltipContent>Фильтры</TooltipContent>
      </Tooltip>

      <Credenza open={open} onOpenChange={setOpen}>
        <CredenzaContent>
          <CredenzaHeader>
            <CredenzaTitle>Фильтры</CredenzaTitle>
            <CredenzaDescription>
              Фильтрация заказов по статусу и дате создания.
            </CredenzaDescription>
          </CredenzaHeader>

          <div className="space-y-4">
            <Field>
              <FieldLabel>Статус</FieldLabel>
              <Select
                items={STATUS_ITEMS}
                value={status}
                onValueChange={(next) => onStatusChange((next ?? "all") as OrderStatusFilter)}
              >
                <SelectTrigger className="w-full" aria-label="Статус">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {STATUS_ITEMS.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
            <Field>
              <FieldLabel>Период</FieldLabel>
              <Select
                items={PERIOD_ITEMS}
                value={period}
                onValueChange={(next) => onPeriodChange((next ?? "all") as OrderPeriodFilter)}
              >
                <SelectTrigger className="w-full" aria-label="Период">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {PERIOD_ITEMS.map((item) => (
                      <SelectItem key={item.value} value={item.value}>
                        {item.label}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                </SelectContent>
              </Select>
            </Field>
          </div>

          <div className="flex justify-end gap-2">
            {hasFilters && (
              <Button type="button" variant="ghost" size="lg" onClick={onReset}>
                Сбросить
              </Button>
            )}
            <Button type="button" size="lg" onClick={() => setOpen(false)}>
              Готово
            </Button>
          </div>
        </CredenzaContent>
      </Credenza>
    </>
  );
};

import { useState } from "react";
import { Filter } from "iconsax-reactjs";
import type { ProductStatusFilter, StockLabel } from "@/entities/product";
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
import { CategoryFilter } from "./category-filter";

export type StockFilter = "all" | StockLabel;

const STATUS_ITEMS: { value: ProductStatusFilter; label: string }[] = [
  { value: "all", label: "Все" },
  { value: "active", label: "Активные" },
  { value: "draft", label: "Черновики" },
  { value: "archived", label: "В архиве" },
];

const STOCK_ITEMS: { value: StockFilter; label: string }[] = [
  { value: "all", label: "Любое" },
  { value: "in_stock", label: "В наличии" },
  { value: "low_stock", label: "Мало" },
  { value: "out_of_stock", label: "Нет в наличии" },
];

export type ProductFiltersButtonProps = {
  category: number | null;
  status: ProductStatusFilter;
  stock: StockFilter;
  onCategoryChange: (next: number | null) => void;
  onStatusChange: (next: ProductStatusFilter) => void;
  onStockChange: (next: StockFilter) => void;
  onReset: () => void;
};

export const ProductFiltersButton = ({
  category,
  status,
  stock,
  onCategoryChange,
  onStatusChange,
  onStockChange,
  onReset,
}: ProductFiltersButtonProps) => {
  const [open, setOpen] = useState(false);
  const hasFilters = category !== null || status !== "all" || stock !== "all";

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
              aria-label="Фильтры товаров"
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
              Фильтрация товаров по категории, статусу и наличию на складе.
            </CredenzaDescription>
          </CredenzaHeader>

          <div className="space-y-4">
            <Field>
              <FieldLabel>Категория</FieldLabel>
              <CategoryFilter
                className="w-full sm:w-full"
                value={category}
                onChange={onCategoryChange}
              />
            </Field>
            <Field>
              <FieldLabel>Статус</FieldLabel>
              <Select
                items={STATUS_ITEMS}
                value={status}
                onValueChange={(next) =>
                  onStatusChange((next ?? "all") as ProductStatusFilter)
                }
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
              <FieldLabel>Наличие</FieldLabel>
              <Select
                items={STOCK_ITEMS}
                value={stock}
                onValueChange={(next) => onStockChange((next ?? "all") as StockFilter)}
              >
                <SelectTrigger className="w-full" aria-label="Наличие">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectGroup>
                    {STOCK_ITEMS.map((item) => (
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

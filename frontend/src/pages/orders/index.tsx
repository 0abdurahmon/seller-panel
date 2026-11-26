import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router";
import { cn } from "cn";
import { useOrderCounts, useOrders, type OrderStatusFilter } from "@/entities/order";
import { AdvanceOrderDialog } from "@/features/order/advance-order";
import { CancelOrderDialog } from "@/features/order/cancel-order";
import { OrderFiltersButton, type OrderPeriodFilter } from "@/features/order/filter-orders";
import { getApiErrorMessage } from "@/shared/api";
import {
  Alert,
  AlertAction,
  AlertDescription,
  AlertTitle,
  Button,
  Tabs,
  TabsCount,
  TabsList,
  TabsTrigger,
} from "@/shared/ui";
import { OrderDetails } from "@/widgets/order-details";
import { OrdersTable } from "@/widgets/orders-table";

const PAGE_SIZE = 20;

const STATUS_TABS: { value: OrderStatusFilter; label: string }[] = [
  { value: "all", label: "Все" },
  { value: "new", label: "Новые" },
  { value: "accepted", label: "Принятые" },
  { value: "packed", label: "Собранные" },
  { value: "handed_to_courier", label: "У курьера" },
  { value: "delivered", label: "Доставленные" },
  { value: "cancelled", label: "Отменённые" },
];

function parseStatus(value: string | null): OrderStatusFilter {
  if (value === "waiting") return "waiting";
  return STATUS_TABS.some((tab) => tab.value === value) ? (value as OrderStatusFilter) : "all";
}

const PERIOD_FILTERS: OrderPeriodFilter[] = ["all", "today", "7d", "30d"];

function parsePeriod(value: string | null): OrderPeriodFilter {
  return PERIOD_FILTERS.includes(value as OrderPeriodFilter) ? (value as OrderPeriodFilter) : "all";
}

function matchesPeriod(createdAt: string, period: OrderPeriodFilter) {
  if (period === "all") return true;
  const created = new Date(createdAt).getTime();
  if (Number.isNaN(created)) return true;
  const now = new Date();
  if (period === "today") {
    const startOfDay = new Date(now.getFullYear(), now.getMonth(), now.getDate()).getTime();
    return created >= startOfDay;
  }
  const days = period === "7d" ? 7 : 30;
  return created >= now.getTime() - days * 86_400_000;
}

function parsePositiveInt(value: string | null) {
  const n = Number(value);
  return value && Number.isInteger(n) && n >= 1 ? n : null;
}

function getScrollParent(node: HTMLElement) {
  for (let el = node.parentElement; el; el = el.parentElement) {
    if (/auto|scroll/.test(getComputedStyle(el).overflowY)) return el;
  }
  return null;
}

export const OrdersPage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const status = parseStatus(searchParams.get("status"));
  const period = parsePeriod(searchParams.get("period"));
  // The open order lives in the URL, so a link to an order works and Back closes it.
  const orderId = parsePositiveInt(searchParams.get("order"));

  function changeStatus(next: OrderStatusFilter) {
    const params = new URLSearchParams(searchParams);
    if (next === "all") params.delete("status");
    else params.set("status", next);
    // Links from the earlier paged list may still carry ?page=.
    params.delete("page");
    setSearchParams(params, { replace: true });
  }

  function changePeriod(next: OrderPeriodFilter) {
    const params = new URLSearchParams(searchParams);
    if (next === "all") params.delete("period");
    else params.set("period", next);
    setSearchParams(params, { replace: true });
  }

  function resetFilters() {
    const params = new URLSearchParams(searchParams);
    params.delete("status");
    params.delete("period");
    params.delete("page");
    setSearchParams(params, { replace: true });
  }

  const openOrder = useCallback(
    (order: { id: number }) => {
      const params = new URLSearchParams(searchParams);
      params.set("order", String(order.id));
      setSearchParams(params);
    },
    [searchParams, setSearchParams],
  );

  const closeOrder = useCallback(() => {
    const params = new URLSearchParams(searchParams);
    params.delete("order");
    setSearchParams(params, { replace: true });
  }, [searchParams, setSearchParams]);

  const orders = useOrders({
    page_size: PAGE_SIZE,
    status,
  });
  const counts = useOrderCounts();

  const stickySentinel = useRef<HTMLDivElement>(null);
  const [isTabsStuck, setIsTabsStuck] = useState(false);

  useEffect(() => {
    const node = stickySentinel.current;
    if (!node) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]) setIsTabsStuck(!entries[0].isIntersecting);
      },
      { root: getScrollParent(node), threshold: 0 },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, []);

  // The next page is requested while the sentinel is still 800px below the screen,
  // so by the time the reader gets there the rows are already in.
  const sentinel = useRef<HTMLDivElement>(null);
  const { hasNextPage, isFetchingNextPage, isFetchNextPageError, fetchNextPage } = orders;
  // The last page is usually shorter than a full one: show as many rows as are really coming.
  const loadingMore =
    isFetchingNextPage && orders.data
      ? Math.max(0, Math.min(PAGE_SIZE, orders.data.total - orders.data.items.length))
      : 0;
  useEffect(() => {
    const node = sentinel.current;
    if (!node || !hasNextPage || isFetchingNextPage || isFetchNextPageError) return;
    // rootMargin only widens the root, so the root must be the element that actually scrolls.
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) void fetchNextPage();
      },
      { root: getScrollParent(node), rootMargin: "0px 0px 800px 0px" },
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [hasNextPage, isFetchingNextPage, isFetchNextPageError, fetchNextPage]);

  return (
    <div className="flex flex-1 flex-col gap-6">
      <AdvanceOrderDialog />
      <CancelOrderDialog />
      <OrderDetails orderId={orderId} onClose={closeOrder} />

      <div ref={stickySentinel} className="-mb-6 h-px" aria-hidden />
      <div
        className={cn(
          "sticky -top-3 z-20 -mx-5 -mt-6 flex flex-wrap items-center justify-between gap-4 border-b bg-background px-5 py-3 transition-[box-shadow,border-color] duration-200",
          isTabsStuck
            ? "border-border/60 shadow-lg shadow-foreground/5"
            : "border-transparent",
        )}
      >
        <Tabs
          value={status === "waiting" ? "" : status}
          onValueChange={(value) => changeStatus(value as OrderStatusFilter)}
        >
          <TabsList>
            {STATUS_TABS.map((tab) => (
              <TabsTrigger key={tab.value} value={tab.value}>
                {tab.label} <TabsCount count={counts[tab.value]} />
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>

        <OrderFiltersButton
          status={status}
          period={period}
          onStatusChange={changeStatus}
          onPeriodChange={changePeriod}
          onReset={resetFilters}
        />
      </div>

      {/* Only when there is nothing to show: a failed next page or background refetch keeps the loaded rows. */}
      {orders.isError && !orders.data ? (
        <Alert variant="destructive">
          <AlertTitle>Не удалось загрузить заказы</AlertTitle>
          <AlertDescription>{getApiErrorMessage(orders.error)}</AlertDescription>
          <AlertAction>
            <Button size="sm" variant="outline" onClick={() => orders.refetch()}>
              Повторить
            </Button>
          </AlertAction>
        </Alert>
      ) : (
        <>
          <OrdersTable
            orders={
              orders.data?.items && period !== "all"
                ? orders.data.items.filter((item) => matchesPeriod(item.created_at, period))
                : orders.data?.items
            }
            isLoading={orders.isPending}
            loadingMore={loadingMore}
            status={status}
            onOpenOrder={openOrder}
          />
          {orders.isFetchNextPageError ? (
            <div className="flex items-center justify-center gap-3 py-2 text-sm text-muted-foreground">
              Не удалось загрузить следующие заказы.
              <Button size="sm" variant="outline" onClick={() => void fetchNextPage()}>
                Повторить
              </Button>
            </div>
          ) : (
            <>
              <div ref={sentinel} className="h-px" />
              {/* The skeleton rows are only for the eyes; this says the same to a screen reader. */}
              <p role="status" className="sr-only">
                {loadingMore > 0 && `Загружаем ещё заказы: ${loadingMore}`}
              </p>
            </>
          )}
        </>
      )}
    </div>
  );
};

import { useCallback } from "react";
import { useSearchParams } from "react-router";
import { useOrderCounts, useOrders, type OrderStatusFilter } from "@/entities/order";
import { AdvanceOrderDialog } from "@/features/order/advance-order";
import { CancelOrderDialog } from "@/features/order/cancel-order";
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
import { OrdersPagination, OrdersTable } from "@/widgets/orders-table";

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
  return STATUS_TABS.some((tab) => tab.value === value) ? (value as OrderStatusFilter) : "all";
}

function parsePositiveInt(value: string | null) {
  const n = Number(value);
  return value && Number.isInteger(n) && n >= 1 ? n : null;
}

export const OrdersPage = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const status = parseStatus(searchParams.get("status"));
  const page = parsePositiveInt(searchParams.get("page")) ?? 1;
  // The open order lives in the URL, so a link to an order works and Back closes it.
  const orderId = parsePositiveInt(searchParams.get("order"));

  function updateParams(next: { status?: OrderStatusFilter; page?: number }) {
    const params = new URLSearchParams(searchParams);
    if (next.status !== undefined) {
      if (next.status === "all") params.delete("status");
      else params.set("status", next.status);
    }
    // Any new filter starts from the first page; only an explicit page change keeps one.
    if (next.page && next.page > 1) params.set("page", String(next.page));
    else params.delete("page");
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
    page,
    page_size: PAGE_SIZE,
    ...(status === "all" ? {} : { status }),
  });
  const counts = useOrderCounts();

  return (
    <div className="flex flex-1 flex-col gap-6">
      <AdvanceOrderDialog />
      <CancelOrderDialog />
      <OrderDetails orderId={orderId} onClose={closeOrder} />

      <Tabs value={status} onValueChange={(value) => updateParams({ status: value as OrderStatusFilter })}>
        <TabsList>
          {STATUS_TABS.map((tab) => (
            <TabsTrigger key={tab.value} value={tab.value}>
              {tab.label} <TabsCount count={counts[tab.value]} />
            </TabsTrigger>
          ))}
        </TabsList>
      </Tabs>

      {orders.isError ? (
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
            orders={orders.data?.items}
            isLoading={orders.isPending}
            status={status}
            onOpenOrder={openOrder}
          />
          {orders.data && (
            <OrdersPagination
              page={orders.data.page}
              pageSize={orders.data.page_size}
              total={orders.data.total}
              disabled={orders.isPlaceholderData}
              onPageChange={(next) => updateParams({ page: next })}
            />
          )}
        </>
      )}
    </div>
  );
};

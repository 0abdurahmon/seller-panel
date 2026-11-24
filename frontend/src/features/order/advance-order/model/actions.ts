import { Box, TickCircle, TruckFast, Verify } from "iconsax-reactjs";
import type { OrderStatus } from "@/entities/order";

type AdvanceAction = {
  /** The button and the menu item. */
  label: string;
  /** The confirm button. */
  confirmLabel: string;
  title: (number: string) => string;
  /** What the change does, so the seller knows before confirming. */
  description: string;
  icon: typeof TickCircle;
};

// Keyed by the status the order moves to.
export const ADVANCE_ACTIONS: Partial<Record<OrderStatus, AdvanceAction>> = {
  accepted: {
    label: "Принять",
    confirmLabel: "Принять",
    title: (number) => `Принять заказ ${number}?`,
    description:
      "Товары заказа будут зарезервированы на складе. Если свободного остатка не хватит, принять заказ не получится.",
    icon: TickCircle,
  },
  packed: {
    label: "Отметить собранным",
    confirmLabel: "Заказ собран",
    title: (number) => `Заказ ${number} собран?`,
    description: "Заказ собран и ждёт курьера.",
    icon: Box,
  },
  handed_to_courier: {
    label: "Передать курьеру",
    confirmLabel: "Передать",
    title: (number) => `Передать заказ ${number} курьеру?`,
    description: "Заказ уходит к курьеру. После этого его нельзя отменить.",
    icon: TruckFast,
  },
  delivered: {
    label: "Отметить доставленным",
    confirmLabel: "Заказ доставлен",
    title: (number) => `Заказ ${number} доставлен?`,
    description:
      "Товары спишутся со склада, а выручка и комиссия попадут в баланс. Отменить это нельзя.",
    icon: Verify,
  },
};

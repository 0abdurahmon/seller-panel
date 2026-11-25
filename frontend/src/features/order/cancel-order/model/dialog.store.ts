import { create } from "zustand";
import type { OrderTarget } from "@/entities/order";

interface CancelOrderState {
  /** The order to cancel; null when the dialog is closed. */
  order: OrderTarget | null;
  openOrder: (order: OrderTarget) => void;
  close: () => void;
}

export const useCancelOrder = create<CancelOrderState>((set) => ({
  order: null,
  // Only what the dialog needs, so a list row and a full order both fit.
  openOrder: (order) => set({ order: { id: order.id, number: order.number, status: order.status } }),
  close: () => set({ order: null }),
}));

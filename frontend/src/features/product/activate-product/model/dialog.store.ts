import { create } from "zustand";
import type { Product } from "@/entities/product";

interface ActivateProductState {
  /** The product to activate; null when the dialog is closed. */
  product: Product | null;
  openProduct: (product: Product) => void;
  close: () => void;
}

export const useActivateProduct = create<ActivateProductState>((set) => ({
  product: null,
  openProduct: (product) => set({ product }),
  close: () => set({ product: null }),
}));

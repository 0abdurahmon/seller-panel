import { create } from "zustand";
import type { Product } from "@/entities/product";

interface RestockProductState {
  /** The product whose stock is being updated; null when closed. */
  product: Product | null;
  openProduct: (product: Product) => void;
  close: () => void;
}

export const useRestockProduct = create<RestockProductState>((set) => ({
  product: null,
  openProduct: (product) => set({ product }),
  close: () => set({ product: null }),
}));

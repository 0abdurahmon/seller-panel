import { create } from "zustand";
import type { Product } from "@/entities/product";

interface ProductDetailsState {
  /** The product shown in the modal; null when it is closed. */
  product: Product | null;
  openProduct: (product: Product) => void;
  close: () => void;
}

export const useProductDetails = create<ProductDetailsState>((set) => ({
  product: null,
  openProduct: (product) => set({ product }),
  close: () => set({ product: null }),
}));

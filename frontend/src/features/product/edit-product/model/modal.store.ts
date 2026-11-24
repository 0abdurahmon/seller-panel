import { create } from "zustand";
import type { Product } from "@/entities/product";

interface EditProductState {
  /** The product being edited; null when the modal is closed. */
  product: Product | null;
  openProduct: (product: Product) => void;
  close: () => void;
}

export const useEditProduct = create<EditProductState>((set) => ({
  product: null,
  openProduct: (product) => set({ product }),
  close: () => set({ product: null }),
}));

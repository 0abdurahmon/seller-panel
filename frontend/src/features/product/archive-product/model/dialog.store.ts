import { create } from "zustand";
import type { Product } from "@/entities/product";

interface ArchiveProductState {
  /** The product to archive; null when the dialog is closed. */
  product: Product | null;
  openProduct: (product: Product) => void;
  close: () => void;
}

export const useArchiveProduct = create<ArchiveProductState>((set) => ({
  product: null,
  openProduct: (product) => set({ product }),
  close: () => set({ product: null }),
}));

import { create } from "zustand";

interface CreateProductState {
  open: boolean;
  setOpen: (open: boolean) => void;
}

export const useCreateProduct = create<CreateProductState>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}));

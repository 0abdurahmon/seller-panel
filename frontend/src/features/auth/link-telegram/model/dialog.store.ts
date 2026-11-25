import { create } from 'zustand';

type LinkTelegramDialogState = {
  open: boolean;
  setOpen: (open: boolean) => void;
};

export const useLinkTelegramDialog = create<LinkTelegramDialogState>((set) => ({
  open: false,
  setOpen: (open) => set({ open }),
}));

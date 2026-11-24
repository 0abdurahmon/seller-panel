import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/shared/ui";
import { useCreateProduct } from "../model/modal.store";
import { CreateProductForm } from "./form";

export const CreateProductModal = () => {
  const { open, setOpen } = useCreateProduct();
  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="max-w-2xl!">
        <DialogHeader>
          <DialogTitle>Новый товар</DialogTitle>
          <DialogDescription>
            Заполните данные товара. Их можно будет изменить позже.
          </DialogDescription>
        </DialogHeader>
        {/* Mounted only while open, so the form starts empty every time. */}
        {open && <CreateProductForm onDone={() => setOpen(false)} />}
      </DialogContent>
    </Dialog>
  );
};

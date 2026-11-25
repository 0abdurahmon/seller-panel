import {
  Credenza,
  CredenzaContent,
  CredenzaDescription,
  CredenzaHeader,
  CredenzaTitle,
} from "@/shared/ui";
import { useCreateProduct } from "../model/modal.store";
import { CreateProductForm } from "./form";

export const CreateProductModal = () => {
  const { open, setOpen } = useCreateProduct();
  return (
    <Credenza open={open} onOpenChange={setOpen}>
      <CredenzaContent className="max-w-2xl!">
        <CredenzaHeader>
          <CredenzaTitle>Новый товар</CredenzaTitle>
          <CredenzaDescription>
            Заполните данные товара. Их можно будет изменить позже.
          </CredenzaDescription>
        </CredenzaHeader>
        {/* Mounted only while open, so the form starts empty every time. */}
        {open && <CreateProductForm onDone={() => setOpen(false)} />}
      </CredenzaContent>
    </Credenza>
  );
};

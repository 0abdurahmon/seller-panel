import { Button } from "@/shared/ui";
import { useCreateProduct } from "../model/modal.store";

export const CreateProductButton = () => {
  const { setOpen } = useCreateProduct();
  return (
    <Button onClick={() => setOpen(true)}>
      Добавить товар
    </Button>
  );
};

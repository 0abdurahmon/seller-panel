import { Link } from "react-router";
import { RegisterForm } from "@/features/auth/register";
import { Button } from "@/shared/ui";

export const RegisterPage = () => {
  return (
    <div className="flex flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Откройте магазин в Dona</h1>
        <p className="text-muted-foreground">Аккаунт и магазин создаются вместе, это займёт минуту</p>
      </div>
      <RegisterForm />
      <p className="text-center text-sm text-muted-foreground">
        Уже есть аккаунт?{" "}
        <Button variant="link" className="h-auto p-0" nativeButton={false} render={<Link to="/login" />}>
          Войти
        </Button>
      </p>
    </div>
  );
};

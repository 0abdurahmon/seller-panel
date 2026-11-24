import { Link } from "react-router";
import { LoginForm } from "@/features/auth/login";
import { Button } from "@/shared/ui";

export const LoginPage = () => {
  return (
    <div className="flex flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">С возвращением в Dona</h1>
        <p className="text-muted-foreground">Войдите, чтобы управлять своим магазином</p>
      </div>
      <LoginForm />
      <p className="text-center text-sm text-muted-foreground">
        Нет аккаунта?{" "}
        <Button variant="link" className="h-auto p-0" nativeButton={false} render={<Link to="/register" />}>
          Создать магазин
        </Button>
      </p>
    </div>
  );
};

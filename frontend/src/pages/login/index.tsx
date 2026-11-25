import { Link } from "react-router";
import { LoginForm } from "@/features/auth/login";
import { TelegramLoginForm } from "@/features/auth/telegram-login";
import { Button, Tabs, TabsContent, TabsList, TabsTrigger } from "@/shared/ui";

export const LoginPage = () => {
  return (
    <div className="flex flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">С возвращением в Dona</h1>
        <p className="text-muted-foreground">Войдите, чтобы управлять своим магазином</p>
      </div>
      <Tabs defaultValue="password" className="gap-4">
        <TabsList className="w-full">
          <TabsTrigger value="password" className="flex-1">
            По паролю
          </TabsTrigger>
          <TabsTrigger value="telegram" className="flex-1">
            Через Telegram
          </TabsTrigger>
        </TabsList>
        <TabsContent value="password">
          <LoginForm />
        </TabsContent>
        <TabsContent value="telegram">
          <TelegramLoginForm />
        </TabsContent>
      </Tabs>
      <p className="text-center text-sm text-muted-foreground">
        Нет аккаунта?{" "}
        <Button variant="link" className="h-auto p-0" nativeButton={false} render={<Link to="/register" />}>
          Создать магазин
        </Button>
      </p>
    </div>
  );
};

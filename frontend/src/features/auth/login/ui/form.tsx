import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useLocation, useNavigate } from "react-router";
import { Sms } from "iconsax-reactjs";
import { useSession } from "@/entities/session";
import { getAuth, getApiErrorMessage, type LoginRequest } from "@/shared/api";
import {
  Alert,
  AlertDescription,
  Button,
  Field,
  FieldError,
  FieldLabel,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  PasswordInput,
  Spinner,
} from "@/shared/ui";
import { loginSchema, type LoginValues } from "../model/schema";

const api = getAuth();

/** Where the seller was going before the sign-in page, set by the route guard. */
function useReturnPath() {
  const state = useLocation().state as { from?: string } | null;
  return state?.from ?? "/";
}

export const LoginForm = () => {
  const session = useSession();
  const navigate = useNavigate();
  const returnPath = useReturnPath();

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });

  const login = useMutation({
    mutationFn: (values: LoginRequest) => api.postAuthLogin(values),
    onSuccess: ({ token }) => {
      session.start(token);
      navigate(returnPath, { replace: true });
    },
  });

  const { errors } = form.formState;

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={form.handleSubmit((values) => login.mutate(values))}
      noValidate
    >
      <Controller
        name="email"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.email}>
            <FieldLabel htmlFor="login-email">Email</FieldLabel>
            <InputGroup>
              <InputGroupAddon>
                <Sms />
              </InputGroupAddon>
              <InputGroupInput
                {...field}
                id="login-email"
                type="email"
                autoComplete="email"
                placeholder="you@example.com"
                aria-invalid={!!errors.email}
              />
            </InputGroup>
            <FieldError errors={[errors.email]} />
          </Field>
        )}
      />
      <Controller
        name="password"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.password}>
            <FieldLabel htmlFor="login-password">Пароль</FieldLabel>
            <PasswordInput
              {...field}
              id="login-password"
              autoComplete="current-password"
              placeholder="Ваш пароль"
              aria-invalid={!!errors.password}
            />
            <FieldError errors={[errors.password]} />
          </Field>
        )}
      />

      {login.isError && (
        <Alert variant="destructive">
          <AlertDescription>{getApiErrorMessage(login.error)}</AlertDescription>
        </Alert>
      )}

      <Button type="submit" size="lg" className="mt-2 w-full" disabled={login.isPending}>
        {login.isPending && <Spinner />}
        Войти
      </Button>
    </form>
  );
};

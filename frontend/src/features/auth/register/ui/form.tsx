import { Controller, useForm } from "react-hook-form";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation } from "@tanstack/react-query";
import { useNavigate } from "react-router";
import { Shop, Sms } from "iconsax-reactjs";
import { useSession } from "@/entities/session";
import { getAuth, parseApiError, type RegisterRequest } from "@/shared/api";
import {
  Alert,
  AlertDescription,
  Button,
  Field,
  FieldDescription,
  FieldError,
  FieldLabel,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  PasswordInput,
  Spinner,
} from "@/shared/ui";
import { registerSchema, type RegisterValues } from "../model/schema";

const api = getAuth();

export const RegisterForm = () => {
  const session = useSession();
  const navigate = useNavigate();

  const form = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: { shop_name: "", email: "", password: "" },
  });

  const register = useMutation({
    mutationFn: (values: RegisterRequest) => api.postAuthRegister(values),
    onSuccess: ({ token }) => {
      session.start(token);
      navigate("/", { replace: true });
    },
  });

  const apiError = register.error ? parseApiError(register.error) : null;
  const emailTaken = apiError?.code === "email_taken";
  const { errors } = form.formState;
  const emailError = errors.email ?? (emailTaken ? { message: apiError.message } : undefined);

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={form.handleSubmit((values) => register.mutate(values))}
      noValidate
    >
      <Controller
        name="shop_name"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.shop_name}>
            <FieldLabel htmlFor="register-shop">Название магазина</FieldLabel>
            <InputGroup>
              <InputGroupAddon>
                <Shop />
              </InputGroupAddon>
              <InputGroupInput
                {...field}
                id="register-shop"
                autoComplete="organization"
                placeholder="Например, Gadget Store"
                aria-invalid={!!errors.shop_name}
              />
            </InputGroup>
            <FieldError errors={[errors.shop_name]} />
          </Field>
        )}
      />
      <Controller
        name="email"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!emailError}>
            <FieldLabel htmlFor="register-email">Email</FieldLabel>
            <InputGroup>
              <InputGroupAddon>
                <Sms />
              </InputGroupAddon>
              <InputGroupInput
                {...field}
                id="register-email"
                type="email"
                autoComplete="email"
                placeholder="you@example.com"
                aria-invalid={!!emailError}
              />
            </InputGroup>
            <FieldError errors={[emailError]} />
          </Field>
        )}
      />
      <Controller
        name="password"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.password}>
            <FieldLabel htmlFor="register-password">Пароль</FieldLabel>
            <PasswordInput
              {...field}
              id="register-password"
              autoComplete="new-password"
              placeholder="Придумайте пароль"
              aria-invalid={!!errors.password}
            />
            {errors.password ? (
              <FieldError errors={[errors.password]} />
            ) : (
              <FieldDescription>Минимум 8 символов.</FieldDescription>
            )}
          </Field>
        )}
      />

      {apiError && !emailTaken && (
        <Alert variant="destructive">
          <AlertDescription>{apiError.message}</AlertDescription>
        </Alert>
      )}

      <Button type="submit" size="lg" className="mt-2 w-full" disabled={register.isPending}>
        {register.isPending && <Spinner />}
        Создать магазин
      </Button>
    </form>
  );
};

import { Controller, useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from 'react-router';
import { Sms } from 'iconsax-reactjs';
import { getApiErrorMessage, getAuth, type RequestCodeRequest } from '@/shared/api';
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
  Spinner,
} from '@/shared/ui';
import { requestCodeSchema, type RequestCodeValues } from '../model/schema';

const api = getAuth();

export const TelegramLoginForm = () => {
  const navigate = useNavigate();

  const form = useForm<RequestCodeValues>({
    resolver: zodResolver(requestCodeSchema),
    defaultValues: { email: '' },
  });

  const requestCode = useMutation({
    mutationFn: (values: RequestCodeRequest) => api.postAuthRequestCode(values),
    onSuccess: (_data, variables) => {
      navigate('/login/otp', { state: { email: variables.email } });
    },
  });

  const { errors } = form.formState;

  return (
    <form
      className="flex flex-col gap-4"
      onSubmit={form.handleSubmit((values) => requestCode.mutate(values))}
      noValidate
    >
      <Controller
        name="email"
        control={form.control}
        render={({ field }) => (
          <Field data-invalid={!!errors.email}>
            <FieldLabel htmlFor="telegram-email">Email</FieldLabel>
            <InputGroup>
              <InputGroupAddon>
                <Sms />
              </InputGroupAddon>
              <InputGroupInput
                {...field}
                id="telegram-email"
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

      <p className="text-xs text-muted-foreground">
        Отправим 6-значный код в привязанный к аккаунту Telegram-чат.
      </p>

      {requestCode.isError && (
        <Alert variant="destructive">
          <AlertDescription>{getApiErrorMessage(requestCode.error)}</AlertDescription>
        </Alert>
      )}

      <Button type="submit" size="lg" className="mt-2 w-full" disabled={requestCode.isPending}>
        {requestCode.isPending && <Spinner />}
        Получить код в Telegram
      </Button>
    </form>
  );
};

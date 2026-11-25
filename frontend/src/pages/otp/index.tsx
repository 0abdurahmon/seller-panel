import { Link, Navigate, useLocation } from 'react-router';
import { VerifyOtpForm } from '@/features/auth/verify-otp';
import { Button } from '@/shared/ui';

export const OtpPage = () => {
  const location = useLocation();
  const email = (location.state as { email?: string } | null)?.email;

  if (!email) {
    return <Navigate to="/login" replace />;
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="space-y-1">
        <h1 className="text-2xl font-semibold tracking-tight">Введите код</h1>
        <p className="text-sm text-muted-foreground">
          Отправили 6 цифр в Telegram для <span className="font-medium text-foreground">{email}</span>.{' '}
          <Button
            variant="link"
            className="h-auto p-0 text-sm font-normal text-muted-foreground underline hover:text-foreground"
            nativeButton={false}
            render={<Link to="/login" replace />}
          >
            Сменить
          </Button>
        </p>
      </div>

      <VerifyOtpForm email={email} />
    </div>
  );
};

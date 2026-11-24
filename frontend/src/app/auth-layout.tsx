import { Link, Outlet } from 'react-router';
import { Logo } from '@/shared/ui';
import { AuthPromo } from '@/widgets/auth-promo';

const YEAR = new Date().getFullYear();

/** Layout of the sign-in and sign-up pages: brand panel on the left, the form on the right. */
export const AuthLayout = () => {
  return (
    <div className="flex min-h-dvh bg-muted p-3 sm:p-6">
      <div className="flex flex-1 overflow-hidden rounded-3xl bg-card">
        <AuthPromo className="hidden lg:flex lg:w-2/5" />
        <div className="flex flex-1 flex-col">
          <header className="flex h-24 shrink-0 items-center px-6 sm:px-8">
            <Link
              to="/"
              aria-label="Dona"
              className="rounded-xl outline-none focus-visible:ring-3 focus-visible:ring-ring/30"
            >
              <Logo className="h-8 w-auto" />
            </Link>
          </header>
          <main className="flex flex-1 items-center justify-center px-6 py-8 sm:px-8">
            <div className="w-full max-w-sm">
              <Outlet />
            </div>
          </main>
          <footer className="flex h-24 shrink-0 items-center px-6 text-sm text-muted-foreground sm:px-8">
            © Dona {YEAR}
          </footer>
        </div>
      </div>
    </div>
  );
};

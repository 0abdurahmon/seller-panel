import { Navigate, Outlet, useLocation } from 'react-router';
import { hasSession } from '@/entities/session';

/** Signed-in pages: without a session, go to sign-in and come back here afterwards. */
export const RequireSession = () => {
  const location = useLocation();
  if (!hasSession()) {
    const from = location.pathname + location.search;
    return <Navigate to="/login" replace state={{ from }} />;
  }
  return <Outlet />;
};

/** Sign-in and sign-up: a seller who is already signed in goes straight to the panel. */
export const GuestOnly = () => {
  if (hasSession()) return <Navigate to="/" replace />;
  return <Outlet />;
};

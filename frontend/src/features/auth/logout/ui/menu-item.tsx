import { useNavigate } from "react-router";
import { LogoutCurve } from "iconsax-reactjs";
import { useSession } from "@/entities/session";
import { DropdownMenuItem } from "@/shared/ui";

/**
 * Signs out on this device: the token is dropped and the cache cleared.
 * The API has no logout endpoint (D2), so there is nothing to call.
 */
export const LogoutMenuItem = () => {
  const session = useSession();
  const navigate = useNavigate();

  return (
    <DropdownMenuItem
      variant="destructive"
      onClick={() => {
        session.end();
        navigate("/login", { replace: true });
      }}
    >
      <LogoutCurve />
      Выйти
    </DropdownMenuItem>
  );
};

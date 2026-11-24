import { useNavigate } from "react-router";
import { LogoutCurve } from "iconsax-reactjs";
import { useSession } from "@/entities/session";
import { Button } from "@/shared/ui";

/**
 * Signs out on this device: the token is dropped and the cache cleared.
 * The API has no logout endpoint (D2), so there is nothing to call.
 */
export const LogoutButton = ({ collapsed = false }: { collapsed?: boolean }) => {
  const session = useSession();
  const navigate = useNavigate();

  return (
    <Button
      variant="ghost"
      size={collapsed ? "icon-lg" : "lg"}
      aria-label={collapsed ? "Выйти" : undefined}
      className="w-full justify-start rounded-xl px-3.5 text-muted-foreground"
      onClick={() => {
        session.end();
        navigate("/login", { replace: true });
      }}
    >
      <LogoutCurve className="size-5" />
      {!collapsed && "Выйти"}
    </Button>
  );
};

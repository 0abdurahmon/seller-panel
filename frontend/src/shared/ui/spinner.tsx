import { cn } from "cn"
import { LoaderIcon } from "./icons"

function Spinner({ className, ...props }: React.ComponentProps<"svg">) {
  return (
    <LoaderIcon data-slot="spinner" role="status" aria-label="Загрузка" className={cn("size-4 animate-spin", className)} {...props} />
  )
}

export { Spinner }

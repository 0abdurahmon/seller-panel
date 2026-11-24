import { cn } from "cn"
import { Refresh2 } from "iconsax-reactjs"

function Spinner({ className, ...props }: React.ComponentProps<"svg">) {
  return (
    <Refresh2 data-slot="spinner" role="status" aria-label="Загрузка" className={cn("size-4 animate-spin", className)} {...props} />
  )
}

export { Spinner }

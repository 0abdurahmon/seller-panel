import * as React from "react"
import { cn } from "cn"
import { Input } from "./input"

/** An input with an icon or a button inside it, on either side. */
function InputGroup({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="input-group"
      className={cn("group/input-group relative flex w-full items-center", className)}
      {...props}
    />
  )
}

/**
 * Something shown inside the input. `start` is decoration (an icon), so clicks pass through
 * to the input; `end` takes clicks, for a clear or show-password button.
 */
function InputGroupAddon({
  className,
  align = "start",
  ...props
}: React.ComponentProps<"div"> & { align?: "start" | "end" }) {
  return (
    <div
      data-slot="input-group-addon"
      data-align={align}
      className={cn(
        "absolute inset-y-0 flex items-center text-muted-foreground [&_svg:not([class*='size-'])]:size-4",
        "data-[align=start]:pointer-events-none data-[align=start]:left-3.5 data-[align=end]:right-1.5",
        className
      )}
      {...props}
    />
  )
}

/** The input of an InputGroup: leaves room for the addons on its sides. */
function InputGroupInput({ className, ...props }: React.ComponentProps<typeof Input>) {
  return (
    <Input
      data-slot="input-group-input"
      className={cn(
        "group-has-data-[align=start]/input-group:pl-10 group-has-data-[align=end]/input-group:pr-11",
        className
      )}
      {...props}
    />
  )
}

export { InputGroup, InputGroupAddon, InputGroupInput }

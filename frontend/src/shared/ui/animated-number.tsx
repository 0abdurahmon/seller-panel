import { TextMorph } from "torph/react"
import { cn } from "cn"

/**
 * A number that morphs digit by digit when it changes (torph), grouped like the rest of the UI
 * ("18 450 000"). Users who prefer reduced motion get the plain change.
 */
function AnimatedNumber({ value, className }: { value: number; className?: string }) {
  return (
    <TextMorph as="span" locale="ru-RU" className={cn("tabular-nums", className)}>
      {value}
    </TextMorph>
  )
}

export { AnimatedNumber }

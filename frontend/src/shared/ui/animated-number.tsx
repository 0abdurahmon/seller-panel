import { useEffect, useState } from "react"
import { TextMorph } from "torph/react"
import { cn } from "cn"

/**
 * A number that morphs digit by digit when it changes (torph), grouped like the rest of the UI
 * ("18 450 000"). Users who prefer reduced motion get the plain change.
 */
function AnimatedNumber({
  value,
  fromZero = false,
  className,
}: {
  value: number
  fromZero?: boolean
  className?: string
}) {
  const [ready, setReady] = useState(!fromZero)

  useEffect(() => {
    if (!fromZero) return
    const id = requestAnimationFrame(() => setReady(true))
    return () => cancelAnimationFrame(id)
  }, [fromZero])

  return (
    <TextMorph as="span" locale="ru-RU" className={cn("tabular-nums", className)}>
      {ready ? value : 0}
    </TextMorph>
  )
}

export { AnimatedNumber }

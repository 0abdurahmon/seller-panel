import type { ReactNode } from "react"
import { Alert, AlertDescription } from "./alert"
import { Button } from "./button"
import {
  Credenza,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
} from "./credenza"
import { Spinner } from "./spinner"

/** A yes/no question before an action, with its pending and error states. */
function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel,
  variant = "default",
  pending = false,
  error,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: ReactNode
  description?: ReactNode
  confirmLabel: string
  variant?: "default" | "destructive"
  pending?: boolean
  error?: string | null
  onConfirm: () => void
}) {
  return (
    <Credenza open={open} onOpenChange={(next) => !pending && onOpenChange(next)}>
      <CredenzaContent className="max-w-md!" showCloseButton={false}>
        <CredenzaHeader>
          <CredenzaTitle>{title}</CredenzaTitle>
          {description && <CredenzaDescription>{description}</CredenzaDescription>}
        </CredenzaHeader>
        {error && (
          <Alert variant="destructive">
            <AlertDescription>{error}</AlertDescription>
          </Alert>
        )}
        <CredenzaFooter>
          <Button variant="ghost" disabled={pending} onClick={() => onOpenChange(false)}>
            Отмена
          </Button>
          <Button variant={variant} disabled={pending} onClick={onConfirm}>
            {pending && <Spinner />}
            {confirmLabel}
          </Button>
        </CredenzaFooter>
      </CredenzaContent>
    </Credenza>
  )
}

export { ConfirmDialog }

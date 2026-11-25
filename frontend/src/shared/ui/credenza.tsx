import * as React from "react"
import { cn } from "cn"
import { useMediaQuery } from "@/shared/lib"
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "./dialog"
import {
  Drawer,
  DrawerClose,
  DrawerContent,
  DrawerDescription,
  DrawerFooter,
  DrawerHeader,
  DrawerTitle,
  DrawerTrigger,
} from "./drawer"

const DESKTOP_QUERY = "(min-width: 768px)"

const CredenzaContext = React.createContext<{ isDesktop: boolean } | null>(null)

function useCredenza() {
  const context = React.useContext(CredenzaContext)
  const isDesktopQuery = useMediaQuery(DESKTOP_QUERY)
  return context ?? { isDesktop: isDesktopQuery }
}

interface RootCredenzaProps {
  children?: React.ReactNode
  open?: boolean
  onOpenChange?: (open: boolean) => void
}

function Credenza({ children, ...props }: RootCredenzaProps) {
  const isDesktop = useMediaQuery(DESKTOP_QUERY)

  return (
    <CredenzaContext.Provider value={{ isDesktop }}>
      {isDesktop ? (
        <Dialog {...props}>{children}</Dialog>
      ) : (
        <Drawer showSwipeHandle {...props}>
          {children}
        </Drawer>
      )}
    </CredenzaContext.Provider>
  )
}

interface CredenzaTriggerProps {
  className?: string
  children?: React.ReactNode
  render?: React.ReactElement
  nativeButton?: boolean
}

function CredenzaTrigger({ className, children, ...props }: CredenzaTriggerProps) {
  const { isDesktop } = useCredenza()

  if (isDesktop) {
    return (
      <DialogTrigger className={className} {...props}>
        {children}
      </DialogTrigger>
    )
  }

  return (
    <DrawerTrigger className={className} {...props}>
      {children}
    </DrawerTrigger>
  )
}

function CredenzaClose({ className, children, ...props }: CredenzaTriggerProps) {
  const { isDesktop } = useCredenza()

  if (isDesktop) {
    return (
      <DialogClose className={className} {...props}>
        {children}
      </DialogClose>
    )
  }

  return (
    <DrawerClose className={className} {...props}>
      {children}
    </DrawerClose>
  )
}

interface CredenzaContentProps {
  className?: string
  children?: React.ReactNode
  showCloseButton?: boolean
}

function CredenzaContent({
  className,
  children,
  showCloseButton,
  ...props
}: CredenzaContentProps) {
  const { isDesktop } = useCredenza()

  if (isDesktop) {
    return (
      <DialogContent className={className} showCloseButton={showCloseButton} {...props}>
        {children}
      </DialogContent>
    )
  }

  return (
    <DrawerContent className={className} {...props}>
      {children}
    </DrawerContent>
  )
}

function CredenzaHeader({ className, ...props }: React.ComponentProps<"div">) {
  const { isDesktop } = useCredenza()
  const Component = isDesktop ? DialogHeader : DrawerHeader
  return <Component className={className} {...props} />
}

function CredenzaFooter({
  className,
  showCloseButton,
  ...props
}: React.ComponentProps<typeof DialogFooter>) {
  const { isDesktop } = useCredenza()

  if (isDesktop) {
    return <DialogFooter className={className} showCloseButton={showCloseButton} {...props} />
  }

  return <DrawerFooter className={cn("flex-row justify-end", className)} {...props} />
}

function CredenzaTitle({
  className,
  ...props
}: React.ComponentProps<typeof DialogTitle>) {
  const { isDesktop } = useCredenza()
  const Component = isDesktop ? DialogTitle : DrawerTitle
  return <Component className={className} {...props} />
}

function CredenzaDescription({
  className,
  ...props
}: React.ComponentProps<typeof DialogDescription>) {
  const { isDesktop } = useCredenza()
  const Component = isDesktop ? DialogDescription : DrawerDescription
  return <Component className={className} {...props} />
}

function CredenzaBody({ className, ...props }: React.ComponentProps<"div">) {
  return <div className={cn("min-h-0 flex-1", className)} {...props} />
}

export {
  Credenza,
  CredenzaBody,
  CredenzaClose,
  CredenzaContent,
  CredenzaDescription,
  CredenzaFooter,
  CredenzaHeader,
  CredenzaTitle,
  CredenzaTrigger,
  useCredenza,
}

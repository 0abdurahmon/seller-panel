import type { SVGProps } from "react"

// Primitive icons drawn by hand: iconsax has no plain cross or dots, only boxed and circled ones.
// They follow iconsax's grid (24×24, 1.5 stroke, round caps) and its default 24px size,
// so they sit next to iconsax icons without looking different. Colour is currentColor.

type IconProps = SVGProps<SVGSVGElement>

function IconBase({ children, ...props }: IconProps) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      width="24"
      height="24"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="1.5"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden
      {...props}
    >
      {children}
    </svg>
  )
}

/** A plain cross: close, clear, remove. */
function CloseIcon(props: IconProps) {
  return (
    <IconBase {...props}>
      <path d="M6 6l12 12M18 6L6 18" />
    </IconBase>
  )
}

/** Three dots in a row: more actions, collapsed breadcrumb. */
function MoreHorizontalIcon(props: IconProps) {
  return (
    <IconBase fill="currentColor" stroke="none" {...props}>
      <circle cx="5" cy="12" r="1.75" />
      <circle cx="12" cy="12" r="1.75" />
      <circle cx="19" cy="12" r="1.75" />
    </IconBase>
  )
}

/** Three dots in a column: the actions menu of a table row. */
function MoreVerticalIcon(props: IconProps) {
  return (
    <IconBase fill="currentColor" stroke="none" {...props}>
      <circle cx="12" cy="5" r="1.75" />
      <circle cx="12" cy="12" r="1.75" />
      <circle cx="12" cy="19" r="1.75" />
    </IconBase>
  )
}

export { CloseIcon, MoreHorizontalIcon, MoreVerticalIcon }

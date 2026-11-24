import * as React from "react"
import { Eye, EyeSlash, Lock } from "iconsax-reactjs"
import { Button } from "./button"
import { InputGroup, InputGroupAddon, InputGroupInput } from "./input-group"

/** A password field with a lock icon and a button that shows or hides what is typed. */
function PasswordInput(props: Omit<React.ComponentProps<typeof InputGroupInput>, "type">) {
  const [visible, setVisible] = React.useState(false)

  return (
    <InputGroup>
      <InputGroupAddon>
        <Lock />
      </InputGroupAddon>
      <InputGroupInput type={visible ? "text" : "password"} {...props} />
      <InputGroupAddon align="end">
        <Button
          variant="ghost"
          size="icon-sm"
          className="rounded-full text-muted-foreground"
          aria-label={visible ? "Скрыть пароль" : "Показать пароль"}
          aria-pressed={visible}
          onClick={() => setVisible(!visible)}
        >
          {visible ? <EyeSlash /> : <Eye />}
        </Button>
      </InputGroupAddon>
    </InputGroup>
  )
}

export { PasswordInput }

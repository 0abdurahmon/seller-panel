import { useState, type ChangeEvent, type ComponentProps, type KeyboardEvent } from 'react';
import { formatPhoneNumber, maskPhoneEdit } from '@/shared/lib';
import { Input } from './input';

export function PhoneInput({
  value,
  defaultValue,
  onChange,
  onKeyDown,
  ...props
}: ComponentProps<typeof Input>) {
  const [internalValue, setInternalValue] = useState(() =>
    formatPhoneNumber(String(defaultValue ?? '')),
  );
  const formattedValue = value === undefined ? internalValue : formatPhoneNumber(String(value));

  function handleKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    onKeyDown?.(event);
    if (event.defaultPrevented || event.ctrlKey || event.metaKey || event.altKey) return;
    const input = event.currentTarget;
    const start = input.selectionStart ?? 4;
    const end = input.selectionEnd ?? start;
    if (
      (event.key === 'Backspace' && start <= 4 && end <= 4) ||
      (event.key === 'Delete' && start < 4 && end <= 4)
    ) {
      event.preventDefault();
    } else if (
      start < 4 &&
      (event.key.length === 1 || event.key === 'Backspace' || event.key === 'Delete')
    ) {
      input.setSelectionRange(4, Math.max(4, end));
    }
  }

  function handleChange(event: ChangeEvent<HTMLInputElement>) {
    const input = event.currentTarget;
    const edit = maskPhoneEdit(
      formattedValue,
      input.value,
      input.selectionStart ?? input.value.length,
      (event.nativeEvent as InputEvent).inputType ?? '',
    );
    input.value = edit.value;
    setInternalValue(edit.value);
    onChange?.(event);
    requestAnimationFrame(() => {
      if (document.activeElement === input) input.setSelectionRange(edit.caret, edit.caret);
    });
  }

  return (
    <Input
      {...props}
      type="tel"
      inputMode="tel"
      autoComplete={props.autoComplete ?? 'tel'}
      placeholder={props.placeholder ?? '+998 90 123 45 67'}
      value={formattedValue}
      onChange={handleChange}
      onKeyDown={handleKeyDown}
    />
  );
}

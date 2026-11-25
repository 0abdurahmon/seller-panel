const prefix = '+998';

export function formatPhoneNumber(value: string): string {
  let digits = value.replace(/\D/g, '');
  if (value.trim().startsWith(prefix) || (digits.startsWith('998') && digits.length > 9)) {
    digits = digits.slice(3);
  }
  // A full international number may be pasted after the existing prefix.
  if (digits.startsWith('998') && digits.length > 9) digits = digits.slice(3);
  digits = digits.slice(0, 9);
  const groups = [digits.slice(0, 2), digits.slice(2, 5), digits.slice(5, 7), digits.slice(7, 9)];
  return [prefix, ...groups.filter(Boolean)].join(' ');
}

export function maskPhoneEdit(previous: string, value: string, caret: number, inputType: string) {
  // Deleting a formatting space should also remove the adjacent digit.
  if (formatPhoneNumber(value) === previous && inputType.startsWith('delete')) {
    if (inputType === 'deleteContentBackward') {
      const index = value.slice(0, caret).search(/\d\D*$/);
      if (index > 3) {
        value = value.slice(0, index) + value.slice(index + 1);
        caret = index;
      }
    } else if (inputType === 'deleteContentForward') {
      const offset = value.slice(caret).search(/\d/);
      if (offset >= 0 && caret + offset > 3) {
        const index = caret + offset;
        value = value.slice(0, index) + value.slice(index + 1);
      }
    }
  }

  const formatted = formatPhoneNumber(value);
  const digitsBeforeCaret = value.slice(0, caret).replace(/\D/g, '').length;
  let nextCaret =
    digitsBeforeCaret >= formatted.replace(/\D/g, '').length ? formatted.length : prefix.length;
  let digitsSeen = 0;
  for (let index = 0; index < formatted.length; index++) {
    if (/\d/.test(formatted[index])) digitsSeen++;
    if (digitsSeen === digitsBeforeCaret) {
      nextCaret = Math.max(prefix.length, index + 1);
      break;
    }
  }
  return { value: formatted, caret: nextCaret };
}

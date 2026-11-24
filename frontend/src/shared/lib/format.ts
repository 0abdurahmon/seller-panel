const priceFormat = new Intl.NumberFormat('ru-RU');
const dateTimeFormat = new Intl.DateTimeFormat('ru-RU', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
  hour: '2-digit',
  minute: '2-digit',
});

/** 1500000 -> "1 500 000 сум". Money stays an integer until it is shown. */
export function formatPrice(value: number) {
  return `${priceFormat.format(value)} сум`;
}

export function formatDateTime(value?: string | null) {
  if (!value) return '—';
  return dateTimeFormat.format(new Date(value));
}

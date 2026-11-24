import Axios from 'axios';

const REQUEST_FAILED = 'Не удалось выполнить запрос';

// The backend writes messages in English for developers; the seller sees these instead.
// A code missing here falls back to the backend message.
const MESSAGES: Record<string, string> = {
  invalid_body: 'Запрос составлен неверно. Обновите страницу и попробуйте ещё раз',
  validation_error: 'Проверьте заполненные поля',
  unauthorized: 'Сессия истекла, войдите снова',
  invalid_credentials: 'Неверный email или пароль',
  not_found: 'Не найдено: возможно, запись уже удалена',
  product_not_found: 'Товар не найден: возможно, он уже удалён',
  email_taken: 'Этот email уже зарегистрирован',
  sku_taken: 'Такой артикул уже есть в магазине',
  product_archived: 'Товар в архиве, его нельзя изменить',
  stock_changed: 'Остаток успел измениться. Обновите страницу и попробуйте снова',
  stock_below_reserved: 'Остаток не может быть меньше зарезервированного',
  insufficient_stock: 'Недостаточно товара на складе',
  invalid_transition: 'Такой переход статуса невозможен',
  rate_limited: 'Слишком много запросов. Подождите немного',
  internal: 'Ошибка на сервере. Попробуйте позже',
};

export type ApiErrorData = {
  code?: string;
  message?: string;
  fields?: Record<string, string>;
};

export type ParsedApiError = {
  statusCode?: number;
  code?: string;
  message: string;
  fields: Record<string, string>;
};

export function parseApiError(error: unknown): ParsedApiError {
  if (Axios.isAxiosError<ApiErrorData>(error)) {
    const data = error.response?.data;

    return {
      statusCode: error.response?.status,
      code: data?.code,
      message: (data?.code && MESSAGES[data.code]) || data?.message || REQUEST_FAILED,
      fields: data?.fields ?? {},
    };
  }

  // Not an API answer (network failure, a bug): its own message is technical English.
  return { message: REQUEST_FAILED, fields: {} };
}

export function getApiErrorMessage(error: unknown, fallbackMessage?: string) {
  const parsed = parseApiError(error);
  return parsed.message || fallbackMessage || REQUEST_FAILED;
}

export function getApiFieldError(error: unknown, field: string) {
  return parseApiError(error).fields[field] ?? null;
}

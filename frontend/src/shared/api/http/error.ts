import Axios from 'axios';

const REQUEST_FAILED = 'Не удалось выполнить запрос';

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
      message: data?.message ?? REQUEST_FAILED,
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

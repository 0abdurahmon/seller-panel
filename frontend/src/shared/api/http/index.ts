import Axios, { type AxiosRequestConfig } from "axios";
import { clearAuthToken, readAuthToken } from "@/shared/lib";

// The Vite dev server and nginx both proxy /api to the backend, so no CORS is needed.
const API_URL = import.meta.env.VITE_API_BASE_URL ?? "/api";

export const AXIOS_INSTANCE = Axios.create({
  baseURL: API_URL,
});

AXIOS_INSTANCE.interceptors.request.use((config) => {
  const token = readAuthToken();
  if (token && !config.headers.Authorization) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

const AUTH_ENDPOINTS = ["/auth/login", "/auth/register"];

AXIOS_INSTANCE.interceptors.response.use(
  (res) => res,
  (error) => {
    const status = error.response?.status;
    const url: string = error.config?.url ?? "";
    const isAuthEndpoint = AUTH_ENDPOINTS.some((path) => url.includes(path));

    // An expired or foreign token: drop it and send the seller to the login page.
    if (status === 401 && !isAuthEndpoint) {
      clearAuthToken();
      if (
        typeof window !== "undefined" &&
        window.location.pathname !== "/login"
      ) {
        window.location.href = "/login";
      }
    }

    return Promise.reject(error);
  },
);

export type CustomInstanceConfig = AxiosRequestConfig;

export const customInstance = <T>(
  config: CustomInstanceConfig,
  options?: AxiosRequestConfig,
): Promise<T> => {
  const controller = new AbortController();
  const promise = AXIOS_INSTANCE({
    ...config,
    ...options,
    signal: options?.signal ?? config.signal ?? controller.signal,
  }).then(({ data }) => data);

  // react-query calls cancel() when a query is no longer needed.
  // @ts-expect-error - adding cancel to the promise
  promise.cancel = () => controller.abort();

  return promise;
};

export default customInstance;

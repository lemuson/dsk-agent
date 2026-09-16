
export const USER_API_BASE = '/user-api/api';
export const STAFF_API_BASE = '/staff-api/api';

interface RequestOptions extends RequestInit {
  isStaff?: boolean;
}

export class ApiError extends Error {
  status: number;
  data: any;

  constructor(message: string, status: number, data?: any) {
    super(message);
    this.name = 'ApiError';
    this.status = status;
    this.data = data;
  }
}

export async function request<T = any>(
  path: string,
  options: RequestOptions = {}
): Promise<T> {
  const { isStaff = false, headers, ...rest } = options;
  const base = isStaff ? STAFF_API_BASE : USER_API_BASE;
  const url = path.startsWith('http') ? path : `${base}${path.startsWith('/') ? path : `/${path}`}`;

  const defaultHeaders: HeadersInit = {
    'Content-Type': 'application/json',
    'Accept': 'application/json',
  };

  const response = await fetch(url, {
    ...rest,
    headers: {
      ...defaultHeaders,
      ...headers,
    },
    credentials: 'include',
  });

  if (!response.ok) {
    let errorMsg = `Ошибка HTTP ${response.status}`;
    let errorData = null;
    try {
      const text = await response.text();
      try {
        errorData = JSON.parse(text);
        errorMsg = errorData.message || errorData.error || text || errorMsg;
      } catch {
        errorMsg = text || errorMsg;
      }
    } catch {

    }
    throw new ApiError(errorMsg, response.status, errorData);
  }

  if (response.status === 204) {
    return {} as T;
  }

  const contentType = response.headers.get('content-type') || '';
  if (contentType.includes('application/pdf')) {
    return (await response.blob()) as unknown as T;
  }

  const text = await response.text();
  if (!text || text.trim() === '') {
    return {} as T;
  }

  try {
    return JSON.parse(text) as T;
  } catch {
    return text as unknown as T;
  }
}

export const api = {
  get: <T = any>(path: string, isStaff = false) =>
    request<T>(path, { method: 'GET', isStaff }),

  post: <T = any>(path: string, body?: any, isStaff = false) =>
    request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined,
      isStaff,
    }),

  put: <T = any>(path: string, body?: any, isStaff = false) =>
    request<T>(path, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined,
      isStaff,
    }),

  delete: <T = any>(path: string, isStaff = false) =>
    request<T>(path, { method: 'DELETE', isStaff }),
};

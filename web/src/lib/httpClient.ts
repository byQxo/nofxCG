import axios, {
  AxiosError,
  AxiosInstance,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from 'axios'
import { toast } from 'sonner'

const AUTH_STORAGE_KEY = 'nofx_auth_session'

export interface AuthSession {
  accessToken: string
  refreshToken: string
  remember: boolean
}

interface RetryableRequestConfig extends InternalAxiosRequestConfig {
  _retry?: boolean
  silentError?: boolean
  skipAuthRefresh?: boolean
}

export interface ApiResponse<T = any> {
  success: boolean
  data?: T
  message?: string
  errorKey?: string
  errorParams?: Record<string, string>
  statusCode?: number
}

export class ApiError extends Error {
  errorKey?: string
  errorParams?: Record<string, string>
  statusCode?: number

  constructor(
    message: string,
    errorKey?: string,
    errorParams?: Record<string, string>,
    statusCode?: number
  ) {
    super(message)
    this.name = 'ApiError'
    this.errorKey = errorKey
    this.errorParams = errorParams
    this.statusCode = statusCode
  }
}

let memorySession: AuthSession | null = null
let refreshPromise: Promise<AuthSession | null> | null = null

function readStoredSession(): AuthSession | null {
  if (memorySession) {
    return memorySession
  }

  const raw = localStorage.getItem(AUTH_STORAGE_KEY)
  if (!raw) {
    return null
  }

  try {
    const parsed = JSON.parse(raw) as AuthSession
    if (!parsed?.accessToken || !parsed?.refreshToken) {
      localStorage.removeItem(AUTH_STORAGE_KEY)
      return null
    }
    memorySession = parsed
    return parsed
  } catch {
    localStorage.removeItem(AUTH_STORAGE_KEY)
    return null
  }
}

function persistSession(session: AuthSession | null) {
  memorySession = session
  if (!session) {
    localStorage.removeItem(AUTH_STORAGE_KEY)
    return
  }
  if (session.remember) {
    localStorage.setItem(AUTH_STORAGE_KEY, JSON.stringify(session))
  } else {
    localStorage.removeItem(AUTH_STORAGE_KEY)
  }
}

export function setAuthSession(session: AuthSession): void {
  persistSession(session)
}

export function clearAuthSession(): void {
  persistSession(null)
}

export function loadPersistedAuthSession(): AuthSession | null {
  return readStoredSession()
}

export function getCurrentAuthSession(): AuthSession | null {
  return readStoredSession()
}

function redirectToLogin(): void {
  window.dispatchEvent(new Event('unauthorized'))

  if (!window.location.pathname.includes('/login')) {
    const returnUrl = window.location.pathname + window.location.search
    if (returnUrl !== '/login') {
      sessionStorage.setItem('returnUrl', returnUrl)
    }
    sessionStorage.setItem('from401', 'true')
    window.location.href = '/login'
  }
}

async function refreshAccessToken(): Promise<AuthSession | null> {
  const currentSession = readStoredSession()
  if (!currentSession?.refreshToken) {
    return null
  }

  if (!refreshPromise) {
    refreshPromise = axios
      .post('/api/auth/refresh', {
        refresh_token: currentSession.refreshToken,
      })
      .then((response) => {
        const data = response.data as {
          access_token: string
          refresh_token: string
        }
        const nextSession: AuthSession = {
          accessToken: data.access_token,
          refreshToken: data.refresh_token,
          remember: currentSession.remember,
        }
        setAuthSession(nextSession)
        return nextSession
      })
      .catch(() => {
        clearAuthSession()
        return null
      })
      .finally(() => {
        refreshPromise = null
      })
  }

  return refreshPromise
}

export class HttpClient {
  private axiosInstance: AxiosInstance
  private static isHandlingUnauthorized = false

  constructor() {
    this.axiosInstance = axios.create({
      baseURL: '/',
      timeout: 30000,
      headers: {
        'Content-Type': 'application/json',
      },
    })
    this.setupInterceptors()
  }

  public reset401Flag(): void {
    HttpClient.isHandlingUnauthorized = false
  }

  private setupInterceptors(): void {
    this.axiosInstance.interceptors.request.use(
      (config) => {
        const session = readStoredSession()
        if (session?.accessToken) {
          config.headers = config.headers ?? {}
          config.headers.Authorization = `Bearer ${session.accessToken}`
        }
        return config
      },
      (error) => Promise.reject(error)
    )

    this.axiosInstance.interceptors.response.use(
      (response: AxiosResponse) => response,
      async (error: AxiosError) => this.handleError(error)
    )
  }

  private async handleError(error: AxiosError): Promise<any> {
    const requestConfig = (error.config || {}) as RetryableRequestConfig
    const isSilent = requestConfig.silentError === true
    const status = error.response?.status ?? 0
    const errorData = error.response?.data as
      | {
          error?: string
          message?: string
          error_key?: string
          error_params?: Record<string, string>
        }
      | undefined
    const serverMessage = errorData?.error || errorData?.message

    if (!error.response) {
      if (!isSilent) {
        toast.error('Network error - Please check your connection', {
          id: 'network-error',
          description: 'Unable to reach the server',
        })
      }
      throw new Error('Network error')
    }

    if (
      status === 401 &&
      !requestConfig.skipAuthRefresh &&
      !requestConfig._retry &&
      !String(requestConfig.url || '').includes('/api/auth/login') &&
      !String(requestConfig.url || '').includes('/api/auth/refresh')
    ) {
      requestConfig._retry = true
      const nextSession = await refreshAccessToken()
      if (nextSession?.accessToken) {
        requestConfig.headers = requestConfig.headers ?? {}
        requestConfig.headers.Authorization = `Bearer ${nextSession.accessToken}`
        return this.axiosInstance.request(requestConfig)
      }
    }

    if (status === 401) {
      clearAuthSession()

      if (HttpClient.isHandlingUnauthorized) {
        throw new Error('Session expired')
      }

      HttpClient.isHandlingUnauthorized = true
      redirectToLogin()
      return new Promise(() => {})
    }

    if (status === 403) {
      if (!isSilent) {
        toast.error('Permission Denied', {
          id: 'permission-denied',
          description: 'You do not have permission to access this resource',
        })
      }
      throw new Error('Permission denied')
    }

    if (status === 404) {
      if (!isSilent) {
        toast.error('API Not Found', {
          id: `404-${requestConfig.url || 'unknown'}`,
          description: 'The requested endpoint does not exist (404)',
        })
      }
      throw new Error('API not found')
    }

    if (status >= 500) {
      if (serverMessage) {
        return Promise.reject(error)
      }
      if (!isSilent) {
        toast.error('Server Error', {
          id: 'server-error',
          description: 'Please try again later',
        })
      }
      throw new Error('Server error')
    }

    return Promise.reject(error)
  }

  async request<T = any>(
    url: string,
    options: {
      method?: 'GET' | 'POST' | 'PUT' | 'DELETE' | 'PATCH'
      data?: any
      params?: any
      headers?: Record<string, string>
      silent?: boolean
      skipAuthRefresh?: boolean
    } = {}
  ): Promise<ApiResponse<T>> {
    try {
      const response = await this.axiosInstance.request<T>({
        url,
        method: options.method || 'GET',
        data: options.data,
        params: options.params,
        headers: options.headers,
        ...(options.silent && { silentError: true }),
        ...(options.skipAuthRefresh && { skipAuthRefresh: true }),
      } as RetryableRequestConfig)

      return {
        success: true,
        data: response.data,
        message: (response.data as any)?.message,
      }
    } catch (error) {
      if (axios.isAxiosError(error) && error.response) {
        const errorData = error.response.data as any
        return {
          success: false,
          message: errorData?.error || errorData?.message || 'Operation failed',
          errorKey: errorData?.error_key,
          errorParams: errorData?.error_params,
          statusCode: error.response.status,
        }
      }
      throw error
    }
  }

  async get<T = any>(
    url: string,
    params?: any,
    headers?: Record<string, string>
  ): Promise<ApiResponse<T>> {
    return this.request<T>(url, { method: 'GET', params, headers })
  }

  async post<T = any>(
    url: string,
    data?: any,
    headers?: Record<string, string>
  ): Promise<ApiResponse<T>> {
    return this.request<T>(url, { method: 'POST', data, headers })
  }

  async put<T = any>(
    url: string,
    data?: any,
    headers?: Record<string, string>
  ): Promise<ApiResponse<T>> {
    return this.request<T>(url, { method: 'PUT', data, headers })
  }

  async delete<T = any>(
    url: string,
    headers?: Record<string, string>
  ): Promise<ApiResponse<T>> {
    return this.request<T>(url, { method: 'DELETE', headers })
  }

  async patch<T = any>(
    url: string,
    data?: any,
    headers?: Record<string, string>
  ): Promise<ApiResponse<T>> {
    return this.request<T>(url, { method: 'PATCH', data, headers })
  }
}

export const httpClient = new HttpClient()

export const reset401Flag = () => httpClient.reset401Flag()

import type {
  AIModel,
  BeginnerOnboardingResponse,
  CreateExchangeRequest,
  CurrentBeginnerWalletResponse,
  Exchange,
  ExchangeAccountStateResponse,
  UpdateExchangeConfigRequest,
  UpdateModelConfigRequest,
} from '../../types'
import { API_BASE, httpClient } from './helpers'

export const configApi = {
  async getModelConfigs(): Promise<AIModel[]> {
    const result = await httpClient.get<AIModel[]>(`${API_BASE}/models`)
    if (!result.success) {
      throw new Error(result.message || 'Failed to fetch model configs')
    }
    return Array.isArray(result.data) ? result.data : []
  },

  async getSupportedModels(): Promise<AIModel[]> {
    const result = await httpClient.get<AIModel[]>(
      `${API_BASE}/supported-models`
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Failed to fetch supported models')
    }
    return result.data
  },

  async getPromptTemplates(): Promise<string[]> {
    const res = await fetch(`${API_BASE}/prompt-templates`)
    if (!res.ok) {
      throw new Error('Failed to fetch prompt templates')
    }
    const data = await res.json()
    if (Array.isArray(data.templates)) {
      return data.templates.map((item: { name: string }) => item.name)
    }
    return []
  },

  async updateModelConfigs(request: UpdateModelConfigRequest): Promise<void> {
    const result = await httpClient.put(`${API_BASE}/models`, request)
    if (!result.success) {
      throw new Error(result.message || 'Failed to update model configs')
    }
  },

  async getExchangeConfigs(): Promise<Exchange[]> {
    const result = await httpClient.get<Exchange[]>(`${API_BASE}/exchanges`)
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Failed to fetch exchange configs')
    }
    return result.data
  },

  async getExchangeAccountState(): Promise<ExchangeAccountStateResponse> {
    const result = await httpClient.get<ExchangeAccountStateResponse>(
      `${API_BASE}/exchanges/account-state`
    )
    if (!result.success || !result.data) {
      throw new Error(
        result.message || 'Failed to fetch exchange account states'
      )
    }
    return result.data
  },

  async getSupportedExchanges(): Promise<Exchange[]> {
    const result = await httpClient.get<Exchange[]>(
      `${API_BASE}/supported-exchanges`
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Failed to fetch supported exchanges')
    }
    return result.data
  },

  async updateExchangeConfigs(
    request: UpdateExchangeConfigRequest
  ): Promise<void> {
    const result = await httpClient.put(`${API_BASE}/exchanges`, request)
    if (!result.success) {
      throw new Error(result.message || 'Failed to update exchange configs')
    }
  },

  async createExchange(request: CreateExchangeRequest): Promise<{ id: string }> {
    const result = await httpClient.post<{ id: string }>(
      `${API_BASE}/exchanges`,
      request
    )
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Failed to create exchange account')
    }
    return result.data
  },

  // Deprecated: 保留旧方法名，内部已经收敛为服务端直接加密存储。
  async createExchangeEncrypted(
    request: CreateExchangeRequest
  ): Promise<{ id: string }> {
    return this.createExchange(request)
  },

  async deleteExchange(exchangeId: string): Promise<void> {
    const result = await httpClient.delete(`${API_BASE}/exchanges/${exchangeId}`)
    if (!result.success) {
      throw new Error(result.message || 'Failed to delete exchange account')
    }
  },

  // Deprecated: 保留旧方法名，内部已经收敛为服务端直接加密存储。
  async updateExchangeConfigsEncrypted(
    request: UpdateExchangeConfigRequest
  ): Promise<void> {
    return this.updateExchangeConfigs(request)
  },

  async getServerIP(): Promise<{
    public_ip: string
    message: string
  }> {
    const result = await httpClient.get<{
      public_ip: string
      message: string
    }>(`${API_BASE}/server-ip`)
    if (!result.success || !result.data) {
      throw new Error(result.message || 'Failed to fetch server IP')
    }
    return result.data
  },

  // Deprecated: 旧 onboarding 页面仅作兼容保留，离线模式下禁止再次触发自动钱包流程。
  async prepareBeginnerOnboarding(): Promise<BeginnerOnboardingResponse> {
    throw new Error('Beginner onboarding has been disabled in offline mode')
  },

  // Deprecated: 旧 onboarding 页面仅作兼容保留，离线模式下禁止再次读取自动钱包状态。
  async getCurrentBeginnerWallet(): Promise<CurrentBeginnerWalletResponse> {
    throw new Error('Beginner onboarding has been disabled in offline mode')
  },
}

export interface AIModel {
  id: string
  name: string
  provider: string
  enabled: boolean
  configured?: boolean
  apiKey?: string
  customApiUrl?: string
  customModelName?: string
  walletAddress?: string
  balanceUsdc?: string
}

export interface TelegramConfig {
  configured?: boolean
  token_masked: string
  is_bound: boolean
  bound_chat_id?: number
  model_id?: string
  username?: string
}

export interface Exchange {
  id: string
  exchange_type: string
  account_name: string
  name: string
  type: 'cex' | 'dex'
  enabled: boolean
  configured?: boolean
  apiKey?: string
  secretKey?: string
  passphrase?: string
  testnet?: boolean
  hyperliquidWalletAddr?: string
  asterUser?: string
  asterSigner?: string
  asterPrivateKey?: string
  lighterWalletAddr?: string
  lighterPrivateKey?: string
  lighterApiKeyPrivateKey?: string
  lighterApiKeyIndex?: number
}

export type ExchangeAccountStatus =
  | 'ok'
  | 'disabled'
  | 'missing_credentials'
  | 'invalid_credentials'
  | 'permission_denied'
  | 'unavailable'

export interface ExchangeAccountState {
  exchange_id: string
  status: ExchangeAccountStatus
  display_balance?: string
  asset?: string
  total_equity?: number
  available_balance?: number
  checked_at: string
  error_code?: string
  error_message?: string
}

export interface ExchangeAccountStateResponse {
  states: Record<string, ExchangeAccountState>
}

export interface CreateExchangeRequest {
  exchange_type: string
  account_name: string
  enabled: boolean
  api_key?: string
  secret_key?: string
  passphrase?: string
  testnet?: boolean
  hyperliquid_wallet_addr?: string
  aster_user?: string
  aster_signer?: string
  aster_private_key?: string
  lighter_wallet_addr?: string
  lighter_private_key?: string
  lighter_api_key_private_key?: string
  lighter_api_key_index?: number
}

export interface CreateTraderRequest {
  name: string
  ai_model_id: string
  exchange_id: string
  strategy_id?: string
  initial_balance?: number
  scan_interval_minutes?: number
  is_cross_margin?: boolean
  show_in_competition?: boolean
  btc_eth_leverage?: number
  altcoin_leverage?: number
  trading_symbols?: string
  custom_prompt?: string
  override_base_prompt?: boolean
  system_prompt_template?: string
  use_ai500?: boolean
  use_oi_top?: boolean
}

export interface UpdateModelConfigRequest {
  models: {
    [key: string]: {
      enabled: boolean
      api_key: string
      custom_api_url?: string
      custom_model_name?: string
    }
  }
}

export interface UpdateExchangeConfigRequest {
  exchanges: {
    [key: string]: {
      enabled: boolean
      api_key: string
      secret_key: string
      passphrase?: string
      testnet?: boolean
      hyperliquid_wallet_addr?: string
      aster_user?: string
      aster_signer?: string
      aster_private_key?: string
      lighter_wallet_addr?: string
      lighter_private_key?: string
      lighter_api_key_private_key?: string
      lighter_api_key_index?: number
    }
  }
}

// Deprecated: 保留旧 onboarding 类型定义以兼容未移除的页面代码。
export interface BeginnerOnboardingResponse {
  address: string
  private_key: string
  chain: string
  asset: string
  provider: string
  default_model: string
  configured_model_id: string
  balance_usdc: string
  env_saved: boolean
  env_path?: string
  reused_existing: boolean
  env_warning?: string
}

// Deprecated: 保留旧 onboarding 类型定义以兼容未移除的页面代码。
export interface CurrentBeginnerWalletResponse {
  found: boolean
  address?: string
  balance_usdc?: string
  source?: string
  claw402_status?: string
}

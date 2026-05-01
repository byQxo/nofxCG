import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import {
  Building2,
  ChevronRight,
  Cpu,
  MessageCircle,
  Pencil,
  Plus,
  Shield,
  User,
} from 'lucide-react'
import { useAuth } from '../contexts/AuthContext'
import { useLanguage } from '../contexts/LanguageContext'
import { api } from '../lib/api'
import {
  getPostAuthPath,
  getUserMode,
  setUserMode,
  type UserMode,
} from '../lib/onboarding'
import { ExchangeConfigModal } from '../components/trader/ExchangeConfigModal'
import { ModelConfigModal } from '../components/trader/ModelConfigModal'
import { TelegramConfigModal } from '../components/trader/TelegramConfigModal'
import type { AIModel, Exchange } from '../types'

type Tab = 'account' | 'models' | 'exchanges' | 'telegram'

function configBadge(label: string, active: boolean) {
  return (
    <span
      className={`text-[11px] px-2 py-0.5 rounded-full ${
        active
          ? 'bg-emerald-500/10 text-emerald-300'
          : 'bg-zinc-800 text-zinc-500'
      }`}
    >
      {label}
    </span>
  )
}

export function SettingsPage() {
  const { user } = useAuth()
  const { language } = useLanguage()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<Tab>('account')
  const [userMode, setUserModeState] = useState<UserMode>(
    () => getUserMode() ?? 'advanced'
  )

  const [configuredModels, setConfiguredModels] = useState<AIModel[]>([])
  const [supportedModels, setSupportedModels] = useState<AIModel[]>([])
  const [showModelModal, setShowModelModal] = useState(false)
  const [editingModel, setEditingModel] = useState<string | null>(null)

  const [exchanges, setExchanges] = useState<Exchange[]>([])
  const [showExchangeModal, setShowExchangeModal] = useState(false)
  const [editingExchange, setEditingExchange] = useState<string | null>(null)

  const [showTelegramModal, setShowTelegramModal] = useState(false)

  const isZh = language === 'zh'

  const refreshModelConfigs = async () => {
    const [configs, supported] = await Promise.all([
      api.getModelConfigs(),
      api.getSupportedModels(),
    ])
    setConfiguredModels(configs)
    setSupportedModels(supported)
  }

  const refreshExchangeConfigs = async () => {
    const refreshed = await api.getExchangeConfigs()
    setExchanges(refreshed)
  }

  useEffect(() => {
    if (activeTab === 'models') {
      refreshModelConfigs().catch((error) => {
        toast.error(
          error instanceof Error ? error.message : 'Failed to load AI models'
        )
      })
    }

    if (activeTab === 'exchanges') {
      refreshExchangeConfigs().catch((error) => {
        toast.error(
          error instanceof Error ? error.message : 'Failed to load exchanges'
        )
      })
    }
  }, [activeTab])

  useEffect(() => {
    const handleRefresh = () => {
      refreshModelConfigs().catch(() => {})
      refreshExchangeConfigs().catch(() => {})
    }

    window.addEventListener('agent-config-refresh', handleRefresh)
    return () =>
      window.removeEventListener('agent-config-refresh', handleRefresh)
  }, [])

  const handleSwitchMode = (nextMode: UserMode) => {
    if (nextMode === userMode) {
      return
    }

    setUserMode(nextMode)
    setUserModeState(nextMode)
    toast.success(
      isZh
        ? `已切换到${nextMode === 'beginner' ? '新手模式' : '老手模式'}`
        : nextMode === 'beginner'
          ? 'Switched to beginner mode'
          : 'Switched to advanced mode'
    )
    navigate(getPostAuthPath(nextMode))
  }

  const handleSaveModel = async (
    modelId: string,
    apiKey: string,
    customApiUrl?: string,
    customModelName?: string
  ) => {
    try {
      const existingModel = configuredModels.find(
        (model) => model.id === modelId
      )
      const modelTemplate = supportedModels.find(
        (model) => model.id === modelId
      )
      const modelToUpdate = existingModel || modelTemplate
      if (!modelToUpdate) {
        toast.error(isZh ? '未找到模型配置' : 'Model not found')
        return
      }

      let updatedModels: AIModel[]
      if (existingModel) {
        updatedModels = configuredModels.map((model) =>
          model.id === modelId
            ? {
                ...model,
                apiKey,
                customApiUrl: customApiUrl || '',
                customModelName: customModelName || '',
                enabled: true,
              }
            : model
        )
      } else {
        updatedModels = [
          ...configuredModels,
          {
            ...modelToUpdate,
            apiKey,
            customApiUrl: customApiUrl || '',
            customModelName: customModelName || '',
            enabled: true,
          },
        ]
      }

      const request = {
        models: Object.fromEntries(
          updatedModels.map((model) => [
            model.provider,
            {
              enabled: model.enabled,
              api_key: model.apiKey || '',
              custom_api_url: model.customApiUrl || '',
              custom_model_name: model.customModelName || '',
            },
          ])
        ),
      }

      await api.updateModelConfigs(request)
      await refreshModelConfigs()
      setShowModelModal(false)
      setEditingModel(null)
      toast.success(isZh ? '模型配置已保存' : 'Model configuration saved')
    } catch {
      toast.error(isZh ? '保存模型配置失败' : 'Failed to save model config')
    }
  }

  const handleDeleteModel = async (modelId: string) => {
    try {
      const updatedModels = configuredModels.map((model) =>
        model.id === modelId
          ? {
              ...model,
              apiKey: '',
              customApiUrl: '',
              customModelName: '',
              enabled: false,
            }
          : model
      )

      const request = {
        models: Object.fromEntries(
          updatedModels.map((model) => [
            model.provider,
            {
              enabled: model.enabled,
              api_key: model.apiKey || '',
              custom_api_url: model.customApiUrl || '',
              custom_model_name: model.customModelName || '',
            },
          ])
        ),
      }

      await api.updateModelConfigs(request)
      await refreshModelConfigs()
      setShowModelModal(false)
      setEditingModel(null)
      toast.success(isZh ? '模型配置已移除' : 'Model configuration removed')
    } catch {
      toast.error(isZh ? '移除模型配置失败' : 'Failed to remove model config')
    }
  }

  const handleSaveExchange = async (
    exchangeId: string | null,
    exchangeType: string,
    accountName: string,
    apiKey: string,
    secretKey?: string,
    passphrase?: string,
    testnet?: boolean,
    hyperliquidWalletAddr?: string,
    asterUser?: string,
    asterSigner?: string,
    asterPrivateKey?: string,
    lighterWalletAddr?: string,
    lighterPrivateKey?: string,
    lighterApiKeyPrivateKey?: string,
    lighterApiKeyIndex?: number
  ) => {
    try {
      if (exchangeId) {
        await api.updateExchangeConfigsEncrypted({
          exchanges: {
            [exchangeId]: {
              enabled: true,
              api_key: apiKey || '',
              secret_key: secretKey || '',
              passphrase: passphrase || '',
              testnet: testnet || false,
              hyperliquid_wallet_addr: hyperliquidWalletAddr || '',
              aster_user: asterUser || '',
              aster_signer: asterSigner || '',
              aster_private_key: asterPrivateKey || '',
              lighter_wallet_addr: lighterWalletAddr || '',
              lighter_private_key: lighterPrivateKey || '',
              lighter_api_key_private_key: lighterApiKeyPrivateKey || '',
              lighter_api_key_index: lighterApiKeyIndex || 0,
            },
          },
        })
      } else {
        await api.createExchangeEncrypted({
          exchange_type: exchangeType,
          account_name: accountName,
          enabled: true,
          api_key: apiKey || '',
          secret_key: secretKey || '',
          passphrase: passphrase || '',
          testnet: testnet || false,
          hyperliquid_wallet_addr: hyperliquidWalletAddr || '',
          aster_user: asterUser || '',
          aster_signer: asterSigner || '',
          aster_private_key: asterPrivateKey || '',
          lighter_wallet_addr: lighterWalletAddr || '',
          lighter_private_key: lighterPrivateKey || '',
          lighter_api_key_private_key: lighterApiKeyPrivateKey || '',
          lighter_api_key_index: lighterApiKeyIndex || 0,
        })
      }

      await refreshExchangeConfigs()
      setShowExchangeModal(false)
      setEditingExchange(null)
      toast.success(isZh ? '交易所配置已保存' : 'Exchange configuration saved')
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : isZh
            ? '保存交易所配置失败'
            : 'Failed to save exchange config'
      )
    }
  }

  const handleDeleteExchange = async (exchangeId: string) => {
    try {
      await api.deleteExchange(exchangeId)
      await refreshExchangeConfigs()
      setShowExchangeModal(false)
      setEditingExchange(null)
      toast.success(isZh ? '交易所账户已删除' : 'Exchange account deleted')
    } catch (error) {
      toast.error(
        error instanceof Error
          ? error.message
          : isZh
            ? '删除交易所账户失败'
            : 'Failed to delete exchange account'
      )
    }
  }

  const tabs: { key: Tab; label: string; icon: React.ReactNode }[] = [
    {
      key: 'account',
      label: isZh ? '账户' : 'Account',
      icon: <User size={16} />,
    },
    {
      key: 'models',
      label: isZh ? 'AI 模型' : 'AI Models',
      icon: <Cpu size={16} />,
    },
    {
      key: 'exchanges',
      label: isZh ? '交易所' : 'Exchanges',
      icon: <Building2 size={16} />,
    },
    {
      key: 'telegram',
      label: 'Telegram',
      icon: <MessageCircle size={16} />,
    },
  ]

  return (
    <div
      className="min-h-screen px-4 pb-12 pt-20"
      style={{ background: '#0B0E11' }}
    >
      <div className="mx-auto max-w-2xl">
        <h1 className="mb-6 text-xl font-bold text-white">
          {isZh ? '设置' : 'Settings'}
        </h1>

        <div className="mb-6 flex gap-1 rounded-xl border border-zinc-800 bg-zinc-900/60 p-1">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`flex flex-1 items-center justify-center gap-2 rounded-lg px-3 py-2 text-sm font-medium transition-all ${
                activeTab === tab.key
                  ? 'bg-nofx-gold text-black'
                  : 'text-zinc-400 hover:text-white'
              }`}
            >
              {tab.icon}
              <span className="hidden sm:inline">{tab.label}</span>
            </button>
          ))}
        </div>

        <div className="rounded-2xl border border-zinc-800/80 bg-zinc-900/60 p-6 backdrop-blur-xl">
          {activeTab === 'account' ? (
            <div className="space-y-6">
              <div className="rounded-2xl border border-zinc-800 bg-zinc-950/60 p-5">
                <div className="mb-3 flex items-center gap-3">
                  <div className="rounded-xl bg-nofx-gold/10 p-2 text-nofx-gold">
                    <Shield size={18} />
                  </div>
                  <div>
                    <div className="text-sm font-semibold text-white">
                      {isZh ? '本地离线管理员模式' : 'Offline Admin Mode'}
                    </div>
                    <div className="text-xs text-zinc-500">
                      {user?.email || 'Local Admin'}
                    </div>
                  </div>
                </div>
                <div className="space-y-2 text-sm text-zinc-400">
                  <p>
                    {isZh
                      ? '当前实例使用管理员密钥登录，不再提供账号密码注册和修改密码功能。'
                      : 'This instance now uses an offline admin key. Email/password flows are disabled.'}
                  </p>
                  <p>
                    {isZh ? '重置管理员密钥：' : 'Reset admin key:'}{' '}
                    <code>./nofx reset-admin-key</code>
                  </p>
                  <p>
                    {isZh
                      ? '轮换根密钥并重加密数据：'
                      : 'Rotate root key and re-encrypt data:'}{' '}
                    <code>./nofx reset-root-key</code>
                  </p>
                  <p>
                    {isZh ? '恢复备份：' : 'Restore backup:'}{' '}
                    <code>./nofx restore-backup &lt;timestamp&gt;</code>
                  </p>
                </div>
              </div>

              <div className="border-t border-zinc-800 pt-6">
                <div className="flex items-center justify-between gap-4">
                  <div>
                    <h3 className="text-sm font-semibold text-white">
                      {isZh ? '使用模式' : 'Usage Mode'}
                    </h3>
                    <p className="mt-1 text-xs text-zinc-500">
                      {isZh
                        ? '仅保留模式切换兼容性，旧 onboarding 页面已默认停用。'
                        : 'Mode switching is kept for compatibility. The legacy onboarding flow is disabled.'}
                    </p>
                  </div>
                  <span className="rounded-full border border-nofx-gold/20 bg-nofx-gold/10 px-3 py-1 text-xs font-semibold text-nofx-gold">
                    {userMode === 'beginner'
                      ? isZh
                        ? '当前：新手模式'
                        : 'Current: Beginner'
                      : isZh
                        ? '当前：老手模式'
                        : 'Current: Advanced'}
                  </span>
                </div>

                <div className="mt-4 grid gap-3 sm:grid-cols-2">
                  <button
                    type="button"
                    onClick={() => handleSwitchMode('beginner')}
                    className={`rounded-2xl border px-4 py-4 text-left transition-all ${
                      userMode === 'beginner'
                        ? 'border-nofx-gold bg-nofx-gold/10'
                        : 'border-zinc-800 bg-zinc-950/70 hover:border-zinc-700'
                    }`}
                  >
                    <div className="text-sm font-semibold text-white">
                      {isZh ? '新手模式' : 'Beginner Mode'}
                    </div>
                    <div className="mt-1 text-xs text-zinc-500">
                      {isZh
                        ? '保留旧入口兼容性，但不会再自动生成或展示钱包私钥。'
                        : 'Keeps legacy entry compatibility, but no longer auto-generates wallet secrets.'}
                    </div>
                  </button>

                  <button
                    type="button"
                    onClick={() => handleSwitchMode('advanced')}
                    className={`rounded-2xl border px-4 py-4 text-left transition-all ${
                      userMode === 'advanced'
                        ? 'border-nofx-gold bg-nofx-gold/10'
                        : 'border-zinc-800 bg-zinc-950/70 hover:border-zinc-700'
                    }`}
                  >
                    <div className="text-sm font-semibold text-white">
                      {isZh ? '老手模式' : 'Advanced Mode'}
                    </div>
                    <div className="mt-1 text-xs text-zinc-500">
                      {isZh
                        ? '直接使用原有交易、配置和监控工作流。'
                        : 'Use the original trading, configuration, and monitoring workflow.'}
                    </div>
                  </button>
                </div>
              </div>
            </div>
          ) : null}

          {activeTab === 'models' ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-sm text-zinc-400">
                  {configuredModels.length}{' '}
                  {isZh ? '个模型已配置' : 'configured models'}
                </p>
                <button
                  onClick={() => {
                    setEditingModel(null)
                    setShowModelModal(true)
                  }}
                  className="flex items-center gap-1.5 rounded-lg bg-nofx-gold/10 px-3 py-1.5 text-xs font-medium text-nofx-gold transition-colors hover:bg-nofx-gold/20"
                >
                  <Plus size={14} />
                  {isZh ? '添加模型' : 'Add Model'}
                </button>
              </div>

              {configuredModels.length === 0 ? (
                <div className="py-8 text-center text-sm text-zinc-600">
                  {isZh ? '暂无 AI 模型配置' : 'No AI models configured yet'}
                </div>
              ) : (
                <div className="space-y-2">
                  {configuredModels.map((model) => (
                    <button
                      key={model.id}
                      onClick={() => {
                        setEditingModel(model.id)
                        setShowModelModal(true)
                      }}
                      className="group flex w-full items-center justify-between rounded-xl border border-zinc-700/50 bg-zinc-800/50 px-4 py-3 transition-colors hover:bg-zinc-800"
                    >
                      <div className="flex items-center gap-3">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-zinc-700">
                          <Cpu size={14} className="text-zinc-300" />
                        </div>
                        <div className="text-left">
                          <p className="text-sm font-medium text-white">
                            {model.name}
                          </p>
                          <div className="mt-1 flex flex-wrap items-center gap-1.5">
                            <p className="text-xs text-zinc-500">
                              {model.provider}
                            </p>
                            {configBadge('API Key', !!model.has_api_key)}
                            {model.customModelName
                              ? configBadge('Custom Model', true)
                              : null}
                            {model.customApiUrl
                              ? configBadge('Base URL', true)
                              : null}
                          </div>
                        </div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span
                          className={`rounded-full px-2 py-0.5 text-xs ${
                            model.enabled
                              ? 'bg-emerald-500/10 text-emerald-400'
                              : 'bg-zinc-700 text-zinc-500'
                          }`}
                        >
                          {model.enabled
                            ? isZh
                              ? '已启用'
                              : 'Active'
                            : isZh
                              ? '已停用'
                              : 'Inactive'}
                        </span>
                        <Pencil
                          size={14}
                          className="text-zinc-600 transition-colors group-hover:text-zinc-400"
                        />
                      </div>
                    </button>
                  ))}
                </div>
              )}
            </div>
          ) : null}

          {activeTab === 'exchanges' ? (
            <div className="space-y-4">
              <div className="flex items-center justify-between">
                <p className="text-sm text-zinc-400">
                  {exchanges.length}{' '}
                  {isZh ? '个交易账户' : 'connected accounts'}
                </p>
                <button
                  onClick={() => {
                    setEditingExchange(null)
                    setShowExchangeModal(true)
                  }}
                  className="flex items-center gap-1.5 rounded-lg bg-nofx-gold/10 px-3 py-1.5 text-xs font-medium text-nofx-gold transition-colors hover:bg-nofx-gold/20"
                >
                  <Plus size={14} />
                  {isZh ? '添加交易所' : 'Add Exchange'}
                </button>
              </div>

              {exchanges.length === 0 ? (
                <div className="py-8 text-center text-sm text-zinc-600">
                  {isZh
                    ? '暂无交易所账户配置'
                    : 'No exchange accounts connected yet'}
                </div>
              ) : (
                <div className="space-y-2">
                  {exchanges.map((exchange) => (
                    <button
                      key={exchange.id}
                      onClick={() => {
                        setEditingExchange(exchange.id)
                        setShowExchangeModal(true)
                      }}
                      className="group flex w-full items-center justify-between rounded-xl border border-zinc-700/50 bg-zinc-800/50 px-4 py-3 transition-colors hover:bg-zinc-800"
                    >
                      <div className="flex items-center gap-3">
                        <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-zinc-700">
                          <Building2 size={14} className="text-zinc-300" />
                        </div>
                        <div className="text-left">
                          <p className="text-sm font-medium text-white">
                            {exchange.account_name || exchange.name}
                          </p>
                          <div className="mt-1 flex flex-wrap items-center gap-1.5">
                            <p className="text-xs capitalize text-zinc-500">
                              {exchange.exchange_type || exchange.type}
                            </p>
                            {configBadge('API Key', !!exchange.has_api_key)}
                            {configBadge('Secret', !!exchange.has_secret_key)}
                            {exchange.has_passphrase
                              ? configBadge('Passphrase', true)
                              : null}
                            {exchange.hyperliquidWalletAddr
                              ? configBadge('Wallet', true)
                              : null}
                            {exchange.has_aster_private_key
                              ? configBadge('Aster Key', true)
                              : null}
                            {exchange.has_lighter_private_key ||
                            exchange.has_lighter_api_key_private_key
                              ? configBadge('Lighter Key', true)
                              : null}
                          </div>
                        </div>
                      </div>
                      <ChevronRight
                        size={14}
                        className="text-zinc-600 transition-colors group-hover:text-zinc-400"
                      />
                    </button>
                  ))}
                </div>
              )}
            </div>
          ) : null}

          {activeTab === 'telegram' ? (
            <div className="space-y-4">
              <p className="text-sm text-zinc-400">
                {isZh
                  ? '连接 Telegram 机器人后，可接收通知并与交易员交互。'
                  : 'Connect a Telegram bot to receive notifications and interact with your traders.'}
              </p>
              <button
                onClick={() => setShowTelegramModal(true)}
                className="group flex w-full items-center justify-between rounded-xl border border-zinc-700/50 bg-zinc-800/50 px-4 py-3 transition-colors hover:bg-zinc-800"
              >
                <div className="flex items-center gap-3">
                  <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-[#0088cc]/20">
                    <MessageCircle size={14} className="text-[#0088cc]" />
                  </div>
                  <span className="text-sm font-medium text-white">
                    {isZh ? '配置 Telegram 机器人' : 'Configure Telegram Bot'}
                  </span>
                </div>
                <ChevronRight
                  size={14}
                  className="text-zinc-600 transition-colors group-hover:text-zinc-400"
                />
              </button>
            </div>
          ) : null}
        </div>
      </div>

      {showModelModal ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 backdrop-blur-sm">
          <ModelConfigModal
            allModels={supportedModels}
            configuredModels={configuredModels}
            editingModelId={editingModel}
            onSave={handleSaveModel}
            onDelete={handleDeleteModel}
            onClose={() => {
              setShowModelModal(false)
              setEditingModel(null)
            }}
            language={language}
          />
        </div>
      ) : null}

      {showExchangeModal ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 backdrop-blur-sm">
          <ExchangeConfigModal
            allExchanges={exchanges}
            editingExchangeId={editingExchange}
            onSave={handleSaveExchange}
            onDelete={handleDeleteExchange}
            onClose={() => {
              setShowExchangeModal(false)
              setEditingExchange(null)
            }}
            language={language}
          />
        </div>
      ) : null}

      {showTelegramModal ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 px-4 backdrop-blur-sm">
          <TelegramConfigModal
            onClose={() => setShowTelegramModal(false)}
            language={language}
          />
        </div>
      ) : null}
    </div>
  )
}

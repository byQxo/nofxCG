import { ArrowRight, Shield, X } from 'lucide-react'
import { useNavigate } from 'react-router-dom'
import { useLanguage } from '../contexts/LanguageContext'
import { markBeginnerOnboardingCompleted } from '../lib/onboarding'

export function BeginnerOnboardingPage() {
  const { language } = useLanguage()
  const navigate = useNavigate()
  const isZh = language === 'zh'

  const handleContinue = () => {
    markBeginnerOnboardingCompleted()
    navigate('/traders')
  }

  return (
    <div className="fixed inset-0 z-[80]">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" />
      <div className="relative flex min-h-screen items-center justify-center px-4 py-10">
        <button
          type="button"
          onClick={handleContinue}
          className="absolute right-6 top-6 inline-flex h-10 w-10 items-center justify-center rounded-full border border-white/10 bg-white/5 text-zinc-400 transition hover:border-white/20 hover:bg-white/10 hover:text-white"
          aria-label={isZh ? '关闭' : 'Close'}
        >
          <X className="h-5 w-5" />
        </button>

        <div className="w-full max-w-2xl rounded-[28px] border border-white/10 bg-[linear-gradient(180deg,rgba(14,18,24,0.96),rgba(8,10,14,0.94))] p-8 shadow-[0_24px_120px_rgba(0,0,0,0.58)]">
          <div className="flex items-center gap-4">
            <div className="flex h-14 w-14 items-center justify-center rounded-[20px] border border-amber-400/20 bg-amber-400/10 text-amber-300">
              <Shield className="h-6 w-6" />
            </div>
            <div>
              <div className="text-xs font-semibold uppercase tracking-[0.28em] text-amber-300/80">
                {isZh ? '兼容保留页面' : 'Compatibility Page'}
              </div>
              <h1 className="mt-2 text-3xl font-bold tracking-tight text-white">
                {isZh
                  ? '新手自动钱包流程已停用'
                  : 'Beginner auto-wallet flow is disabled'}
              </h1>
            </div>
          </div>

          <div className="mt-6 rounded-2xl border border-white/10 bg-white/[0.03] p-5 text-sm leading-7 text-zinc-300">
            <p>
              {isZh
                ? '离线认证改造后，系统不再自动生成、保存或回传钱包私钥。AI 模型与交易所敏感配置只能由你手动填写，并由后端在本地加密存储。'
                : 'After the offline-auth upgrade, the app no longer generates, stores, or returns wallet private keys automatically. Sensitive model and exchange settings must be entered manually and are encrypted locally on the backend.'}
            </p>
            <p className="mt-4">
              {isZh
                ? '如果你是从旧版本升级而来，请直接前往登录页获取管理员密钥登录，然后到 Settings 页面配置 AI 模型、交易所和 Telegram。'
                : 'If you upgraded from an older version, log in with the administrator key first, then configure AI models, exchanges, and Telegram from Settings.'}
            </p>
          </div>

          <div className="mt-6 rounded-2xl border border-blue-500/20 bg-blue-500/10 p-5 text-sm text-blue-100">
            <div className="font-semibold">
              {isZh ? '当前推荐流程' : 'Recommended flow'}
            </div>
            <div className="mt-3 space-y-2 text-blue-100/90">
              <div>
                {isZh
                  ? '1. 在服务端控制台或容器日志中查看管理员登录密钥。'
                  : '1. Read the admin login key from the backend console or container logs.'}
              </div>
              <div>
                {isZh
                  ? '2. 备份 config/keys 目录。'
                  : '2. Back up the config/keys directory.'}
              </div>
              <div>
                {isZh
                  ? '3. 登录后在 Settings 中手动配置敏感参数。'
                  : '3. Log in and configure sensitive settings manually from Settings.'}
              </div>
            </div>
          </div>

          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <button
              type="button"
              onClick={() => navigate('/login')}
              className="inline-flex flex-1 items-center justify-center gap-2 rounded-2xl bg-[linear-gradient(135deg,#2563EB,#1D4ED8)] px-5 py-3 text-sm font-semibold text-white transition hover:scale-[1.01]"
            >
              {isZh ? '前往登录' : 'Go to Login'}
              <ArrowRight className="h-4 w-4" />
            </button>
            <button
              type="button"
              onClick={handleContinue}
              className="inline-flex flex-1 items-center justify-center rounded-2xl border border-white/10 bg-white/5 px-5 py-3 text-sm font-semibold text-zinc-200 transition hover:bg-white/10"
            >
              {isZh ? '返回交易员页面' : 'Back to Traders'}
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

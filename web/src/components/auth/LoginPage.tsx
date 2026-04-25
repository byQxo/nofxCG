import React, { useEffect, useState } from 'react'
import { Eye, EyeOff } from 'lucide-react'
import { toast } from 'sonner'
import { useAuth } from '../../contexts/AuthContext'
import { useLanguage } from '../../contexts/LanguageContext'
import { DeepVoidBackground } from '../common/DeepVoidBackground'
import { LanguageSwitcher } from '../common/LanguageSwitcher'

export function LoginPage() {
  const { language } = useLanguage()
  const { login } = useAuth()
  const [adminKey, setAdminKey] = useState('')
  const [rememberLogin, setRememberLogin] = useState(true)
  const [showSecret, setShowSecret] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')

  const isZh = language === 'zh'

  useEffect(() => {
    if (sessionStorage.getItem('from401') === 'true') {
      toast.warning(
        isZh ? '登录已失效，请重新输入管理员密钥' : 'Session expired, please sign in again'
      )
      sessionStorage.removeItem('from401')
    }
  }, [isZh])

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault()
    setError('')
    setLoading(true)

    const result = await login(adminKey.trim(), rememberLogin)
    setLoading(false)

    if (!result.success) {
      const message =
        result.message ||
        (isZh ? '管理员密钥错误或已被锁定' : 'Admin key is invalid or locked')
      setError(message)
      toast.error(message)
    }
  }

  return (
    <DeepVoidBackground disableAnimation>
      <LanguageSwitcher />

      <div className="flex min-h-screen items-center justify-center px-4 py-16">
        <div className="w-full max-w-md">
          <div className="mb-10 text-center">
            <div className="mb-5 flex justify-center">
              <div className="relative">
                <div className="absolute -inset-3 rounded-full bg-nofx-gold/15 blur-2xl" />
                <img
                  src="/icons/nofx.svg"
                  alt="NOFX"
                  className="relative z-10 h-14 w-14"
                />
              </div>
            </div>
            <h1 className="mb-2 text-3xl font-bold text-white">
              {isZh ? '本地离线登录' : 'Offline Admin Login'}
            </h1>
            <p className="text-sm text-zinc-500">
              {isZh
                ? '管理员密钥已在服务端控制台或容器日志输出'
                : 'The admin key was printed in the server console or container logs'}
            </p>
          </div>

          <div className="rounded-2xl border border-zinc-800/80 bg-zinc-900/60 p-8 shadow-2xl backdrop-blur-xl">
            <form onSubmit={handleSubmit} className="space-y-5">
              <div>
                <label className="mb-2 block text-xs font-medium text-zinc-400">
                  {isZh ? '管理员密钥' : 'Admin Key'}
                </label>
                <div className="relative">
                  <input
                    type={showSecret ? 'text' : 'password'}
                    value={adminKey}
                    onChange={(event) => setAdminKey(event.target.value)}
                    className="w-full rounded-xl border border-zinc-700/80 bg-zinc-950/80 px-4 py-3 pr-11 text-sm text-white placeholder-zinc-600 transition-all focus:border-nofx-gold/60 focus:outline-none focus:ring-1 focus:ring-nofx-gold/30"
                    placeholder={
                      isZh
                        ? '请输入启动时输出的管理员密钥'
                        : 'Enter the admin key printed at startup'
                    }
                    autoFocus
                    required
                  />
                  <button
                    type="button"
                    onClick={() => setShowSecret((value) => !value)}
                    className="absolute right-3.5 top-1/2 -translate-y-1/2 text-zinc-500 transition-colors hover:text-zinc-300"
                  >
                    {showSecret ? <EyeOff size={16} /> : <Eye size={16} />}
                  </button>
                </div>
              </div>

              <label className="flex items-center gap-3 rounded-xl border border-zinc-800 bg-zinc-950/40 px-4 py-3 text-sm text-zinc-300">
                <input
                  type="checkbox"
                  checked={rememberLogin}
                  onChange={(event) => setRememberLogin(event.target.checked)}
                  className="h-4 w-4 rounded border-zinc-600 bg-zinc-900 text-nofx-gold focus:ring-nofx-gold"
                />
                <span>
                  {isZh
                    ? '记住登录状态（保存到 localStorage）'
                    : 'Remember this login on this browser'}
                </span>
              </label>

              {error ? (
                <p className="rounded-lg border border-red-500/20 bg-red-500/10 px-3 py-2 text-xs text-red-400">
                  {error}
                </p>
              ) : null}

              <div className="rounded-xl border border-zinc-800 bg-zinc-950/40 px-4 py-3 text-xs leading-6 text-zinc-400">
                <div>
                  {isZh
                    ? '首次启动后请立即备份 ./config/keys 根密钥目录。'
                    : 'Back up ./config/keys immediately after first startup.'}
                </div>
                <div>
                  {isZh
                    ? '如需重置管理员密钥：./nofx reset-admin-key'
                    : 'Reset admin key: ./nofx reset-admin-key'}
                </div>
                <div>
                  {isZh
                    ? '如需恢复备份：./nofx restore-backup <timestamp>'
                    : 'Restore backup: ./nofx restore-backup <timestamp>'}
                </div>
              </div>

              <button
                type="submit"
                disabled={loading || adminKey.trim() === ''}
                className="mt-2 w-full rounded-xl bg-nofx-gold py-3 text-sm font-semibold text-black transition-all hover:bg-yellow-400 active:scale-[0.98] disabled:cursor-not-allowed disabled:opacity-50"
              >
                {loading
                  ? isZh
                    ? '登录中...'
                    : 'Signing in...'
                  : isZh
                    ? '进入系统'
                    : 'Enter NOFX'}
              </button>
            </form>
          </div>
        </div>
      </div>
    </DeepVoidBackground>
  )
}

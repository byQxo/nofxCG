export interface SystemConfig {
  initialized: boolean
  beta_mode?: boolean
  auth_mode?: string
}

let configPromise: Promise<SystemConfig> | null = null
let cachedConfig: SystemConfig | null = null

export function getSystemConfig(): Promise<SystemConfig> {
  if (cachedConfig) {
    return Promise.resolve(cachedConfig)
  }
  if (configPromise) {
    return configPromise
  }
  configPromise = fetch('/api/config')
    .then(async (res) => {
      if (!res.ok) {
        throw new Error('Failed to load system config')
      }
      return res.json()
    })
    .then((data: SystemConfig) => {
      cachedConfig = data
      return data
    })
  return configPromise
}

/** Call after first-time setup completes so next check reflects initialized=true */
export function invalidateSystemConfig() {
  cachedConfig = null
  configPromise = null
  window.dispatchEvent(new Event('system-config-invalidated'))
}

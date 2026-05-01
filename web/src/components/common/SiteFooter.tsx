import {
  BRAND_INFO,
  DOC_LINKS,
  LICENSE_NOTICE,
  PROJECT_LINKS,
  UPSTREAM_ATTRIBUTION,
} from '../../constants/branding'
import { t, type Language } from '../../i18n/translations'

interface SiteFooterProps {
  language: Language
}

export function SiteFooter({ language }: SiteFooterProps) {
  const links = [
    { label: 'Repository', href: PROJECT_LINKS.repo, accent: '#F0B90B' },
    { label: 'Issues', href: PROJECT_LINKS.issues, accent: '#3B82F6' },
    { label: 'README', href: DOC_LINKS.readmeEn, accent: '#22C55E' },
    { label: '中文 README', href: DOC_LINKS.readmeZh, accent: '#A855F7' },
    { label: 'License', href: DOC_LINKS.license, accent: '#F59E0B' },
    {
      label: 'Disclaimer',
      href: DOC_LINKS.disclaimer,
      accent: '#EF4444',
    },
  ]

  return (
    <footer
      className="mt-16"
      style={{ borderTop: '1px solid #2B3139', background: '#181A20' }}
    >
      <div
        className="max-w-[1920px] mx-auto px-6 py-6 text-center text-sm"
        style={{ color: '#5E6673' }}
      >
        <p>{t('footerTitle', language)}</p>
        <p className="mt-1">{t('footerWarning', language)}</p>
        <p className="mt-2 text-xs">
          {UPSTREAM_ATTRIBUTION} {LICENSE_NOTICE}
        </p>
        <div className="mt-4 flex items-center justify-center gap-3 flex-wrap">
          {links.map((link) => (
            <a
              key={link.label}
              href={link.href}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-3 py-2 rounded text-sm font-semibold transition-all hover:scale-105"
              style={{
                background: '#1E2329',
                color: '#848E9C',
                border: '1px solid #2B3139',
              }}
              onMouseEnter={(e) => {
                e.currentTarget.style.background = '#2B3139'
                e.currentTarget.style.color = '#EAECEF'
                e.currentTarget.style.borderColor = link.accent
              }}
              onMouseLeave={(e) => {
                e.currentTarget.style.background = '#1E2329'
                e.currentTarget.style.color = '#848E9C'
                e.currentTarget.style.borderColor = '#2B3139'
              }}
            >
              {link.label}
            </a>
          ))}
        </div>
        <p className="mt-4 text-xs">{BRAND_INFO.attributionName}</p>
      </div>
    </footer>
  )
}

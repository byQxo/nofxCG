import { motion } from 'framer-motion'
import { CircleHelp, ExternalLink, Github } from 'lucide-react'
import { Language } from '../../i18n/translations'
import { DOC_LINKS, PROJECT_LINKS } from '../../constants/branding'

interface CommunitySectionProps {
  language?: Language
}

export default function CommunitySection({ language }: CommunitySectionProps) {
  return (
    <section className="py-24 relative" style={{ background: '#0B0E11' }}>
      <div
        className="absolute right-0 top-1/2 -translate-y-1/2 w-96 h-96 rounded-full blur-3xl opacity-20"
        style={{
          background:
            'radial-gradient(circle, rgba(240, 185, 11, 0.12) 0%, transparent 70%)',
        }}
      />

      <div className="max-w-6xl mx-auto px-4 relative z-10">
        <motion.div
          className="text-center mb-12"
          initial={{ opacity: 0, y: 30 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
        >
          <h2
            className="text-4xl lg:text-5xl font-bold mb-4"
            style={{ color: '#EAECEF' }}
          >
            {language === 'zh' ? '开源协作入口' : 'Open-Source Entry Points'}
          </h2>
          <p className="text-lg" style={{ color: '#848E9C' }}>
            {language === 'zh'
              ? '查看当前 fork 的仓库、问题追踪与中文文档，不再跳转上游社群。'
              : 'Use this fork’s repository, issue tracker, and README instead of upstream social channels.'}
          </p>
        </motion.div>

        <motion.div
          className="flex flex-wrap items-center justify-center gap-4"
          initial={{ opacity: 0, y: 20 }}
          whileInView={{ opacity: 1, y: 0 }}
          viewport={{ once: true }}
        >
          <a
            href={PROJECT_LINKS.repo}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-xl font-medium transition-all hover:scale-105"
            style={{
              background: 'rgba(240, 185, 11, 0.1)',
              color: '#F0B90B',
              border: '1px solid rgba(240, 185, 11, 0.3)',
            }}
          >
            <Github className="w-5 h-5" />
            {language === 'zh' ? '查看仓库' : 'View Repository'}
          </a>
          <a
            href={PROJECT_LINKS.issues}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-xl font-medium transition-all hover:scale-105"
            style={{
              background: 'rgba(59, 130, 246, 0.1)',
              color: '#60A5FA',
              border: '1px solid rgba(59, 130, 246, 0.3)',
            }}
          >
            <CircleHelp className="w-5 h-5" />
            {language === 'zh' ? '提交问题' : 'Open Issues'}
          </a>
          <a
            href={DOC_LINKS.readmeZh}
            target="_blank"
            rel="noopener noreferrer"
            className="inline-flex items-center gap-2 px-6 py-3 rounded-xl font-medium transition-all hover:scale-105"
            style={{
              background: 'rgba(168, 85, 247, 0.1)',
              color: '#C084FC',
              border: '1px solid rgba(168, 85, 247, 0.3)',
            }}
          >
            <ExternalLink className="w-5 h-5" />
            中文 README
          </a>
        </motion.div>
      </div>
    </section>
  )
}

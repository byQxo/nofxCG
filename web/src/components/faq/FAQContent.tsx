import { useEffect, useRef } from 'react'
import { t, type Language } from '../../i18n/translations'
import type { FAQCategory } from '../../data/faqData'
import { DOC_LINKS, PROJECT_LINKS } from '../../constants/branding'

interface FAQContentProps {
  categories: FAQCategory[]
  language: Language
  onActiveItemChange: (itemId: string) => void
}

function ContributionTasks({ language }: { language: Language }) {
  const isZh = language === 'zh'

  return (
    <div className="space-y-3">
      <div className="text-base">
        {isZh ? '入口：' : 'Entry points:'}{' '}
        <a
          href={PROJECT_LINKS.repo}
          target="_blank"
          rel="noreferrer"
          style={{ color: '#F0B90B' }}
        >
          Repository
        </a>
        {'  |  '}
        <a
          href={PROJECT_LINKS.issues}
          target="_blank"
          rel="noreferrer"
          style={{ color: '#F0B90B' }}
        >
          Issues
        </a>
        {'  |  '}
        <a
          href={DOC_LINKS.readmeZh}
          target="_blank"
          rel="noreferrer"
          style={{ color: '#F0B90B' }}
        >
          中文 README
        </a>
      </div>

      <ol className="list-decimal pl-5 space-y-1 text-base">
        {isZh ? (
          <>
            <li>先阅读当前 fork 的 README、FAQ 和 Issues，确认任务范围与验收标准。</li>
            <li>优先从当前仓库的 labels 中筛选 `good first issue`、`help wanted`、`frontend`、`backend`。</li>
            <li>Fork 当前仓库到你的 GitHub 账户。</li>
            <li>
              从 <code>main</code> 创建功能分支：
              <code className="ml-2">git checkout -b feat/your-topic</code>
            </li>
            <li>
              完成修改后先自检：
              <code className="ml-2">npm --prefix web run lint && npm --prefix web run build</code>
            </li>
            <li>
              推送到你的 fork：
              <code className="ml-2">git push origin feat/your-topic</code>
            </li>
            <li>
              打开 PR，base 选择 <code>byQxo/nofxCG:main</code>，并在描述里关联 issue，例如 <code className="ml-1">Closes #123</code>。
            </li>
          </>
        ) : (
          <>
            <li>Start with this fork’s README, FAQ, and Issues to confirm scope and acceptance criteria.</li>
            <li>Prefer labels such as <code>good first issue</code>, <code>help wanted</code>, <code>frontend</code>, and <code>backend</code> in the current repository.</li>
            <li>Fork the current repository to your GitHub account.</li>
            <li>
              Create a feature branch from <code>main</code>:
              <code className="ml-2">git checkout -b feat/your-topic</code>
            </li>
            <li>
              Run checks before opening a PR:
              <code className="ml-2">npm --prefix web run lint && npm --prefix web run build</code>
            </li>
            <li>
              Push to your fork:
              <code className="ml-2">git push origin feat/your-topic</code>
            </li>
            <li>
              Open a PR targeting <code>byQxo/nofxCG:main</code> and reference the related issue, for example <code className="ml-1">Closes #123</code>.
            </li>
          </>
        )}
      </ol>

      <div
        className="rounded p-3 mt-3"
        style={{
          background: 'rgba(240, 185, 11, 0.08)',
          border: '1px solid rgba(240, 185, 11, 0.25)',
        }}
      >
        {isZh ? (
          <div className="text-sm">
            <strong style={{ color: '#F0B90B' }}>提示：</strong>{' '}
            如果当前仓库启用了 bounty 流程，请优先查看
            <a
              href={PROJECT_LINKS.bountyLabel}
              target="_blank"
              rel="noreferrer"
              style={{ color: '#F0B90B' }}
            >
              {' '}bounty labels
            </a>
            ，并在完成后使用
            <a
              href={PROJECT_LINKS.bountyClaim}
              target="_blank"
              rel="noreferrer"
              style={{ color: '#F0B90B' }}
            >
              {' '}Bounty Claim
            </a>
            模板；如果没有相关任务，就按普通开源贡献流程提交即可。
          </div>
        ) : (
          <div className="text-sm">
            <strong style={{ color: '#F0B90B' }}>Note:</strong>{' '}
            If this fork is using bounty flows, check the
            <a
              href={PROJECT_LINKS.bountyLabel}
              target="_blank"
              rel="noreferrer"
              style={{ color: '#F0B90B' }}
            >
              {' '}bounty labels
            </a>
            and submit the
            <a
              href={PROJECT_LINKS.bountyClaim}
              target="_blank"
              rel="noreferrer"
              style={{ color: '#F0B90B' }}
            >
              {' '}Bounty Claim
            </a>
            template after completion. Otherwise, follow the standard open-source PR flow in this fork.
          </div>
        )}
      </div>
    </div>
  )
}

function PRGuidelines({ language }: { language: Language }) {
  const isZh = language === 'zh'

  return (
    <div className="space-y-3">
      <div className="text-base">
        {isZh ? '参考文档：' : 'References:'}{' '}
        <a
          href={DOC_LINKS.contributing}
          target="_blank"
          rel="noreferrer"
          className="text-nofx-gold hover:underline"
        >
          CONTRIBUTING.md
        </a>
        {'  |  '}
        <a
          href={DOC_LINKS.prGuide}
          target="_blank"
          rel="noreferrer"
          className="text-nofx-gold hover:underline"
        >
          PR_TITLE_GUIDE.md
        </a>
      </div>

      <ol className="list-decimal pl-5 space-y-1 text-base">
        {isZh ? (
          <>
            <li>Fork 当前仓库后，从你的 fork 的 <code>main</code> 创建新分支，不要直接向默认分支提交。</li>
            <li>分支建议使用 <code>feat/</code>、<code>fix/</code>、<code>docs/</code> 等前缀，提交信息遵循 Conventional Commits。</li>
            <li>
              提交前运行检查：
              <code className="ml-2">npm --prefix web run lint && npm --prefix web run build</code>
            </li>
            <li>涉及 UI 变化时附带截图或短视频，降低 review 成本。</li>
            <li>PR 描述里写清影响范围、测试方式、风险点，并关联 issue。</li>
            <li>
              目标仓库选择 <code>byQxo/nofxCG:main</code>，保持 PR 小而聚焦。
            </li>
          </>
        ) : (
          <>
            <li>After forking, branch from your fork’s <code>main</code> and avoid direct commits to the default branch.</li>
            <li>Prefer prefixes such as <code>feat/</code>, <code>fix/</code>, and <code>docs/</code>, and follow Conventional Commits.</li>
            <li>
              Run checks before opening the PR:
              <code className="ml-2">npm --prefix web run lint && npm --prefix web run build</code>
            </li>
            <li>Attach screenshots or a short video for UI changes.</li>
            <li>Summarize scope, testing, and risk in the PR description and link the related issue.</li>
            <li>
              Target <code>byQxo/nofxCG:main</code> and keep PRs small and focused.
            </li>
          </>
        )}
      </ol>

      <div className="rounded p-3 mt-3 bg-nofx-gold/10 border border-nofx-gold/25">
        {isZh ? (
          <div className="text-sm">
            <strong className="text-nofx-gold">提示：</strong>{' '}
            本 fork 会优先通过仓库文档、Issues 和 PR 讨论来协作。涉及奖励或专项任务时，以当前仓库中的
            <a
              href={PROJECT_LINKS.bountyLabel}
              target="_blank"
              rel="noreferrer"
              style={{ color: '#F0B90B' }}
            >
              {' '}bounty labels
            </a>
            与
            <a
              href={PROJECT_LINKS.bountyClaim}
              target="_blank"
              rel="noreferrer"
              style={{ color: '#F0B90B' }}
            >
              {' '}Bounty Claim
            </a>
            模板为准。
          </div>
        ) : (
          <div className="text-sm">
            <strong style={{ color: '#F0B90B' }}>Note:</strong>{' '}
            Collaboration for this fork is repository-first. If bounty or special-task flows are active, use the current repository’s
            <a
              href={PROJECT_LINKS.bountyLabel}
              target="_blank"
              rel="noreferrer"
              style={{ color: '#F0B90B' }}
            >
              {' '}bounty labels
            </a>
            and
            <a
              href={PROJECT_LINKS.bountyClaim}
              target="_blank"
              rel="noreferrer"
              style={{ color: '#F0B90B' }}
            >
              {' '}Bounty Claim
            </a>
            template as the source of truth.
          </div>
        )}
      </div>
    </div>
  )
}

export function FAQContent({
  categories,
  language,
  onActiveItemChange,
}: FAQContentProps) {
  const sectionRefs = useRef<Map<string, HTMLElement>>(new Map())

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => {
        entries.forEach((entry) => {
          if (entry.isIntersecting) {
            const itemId = entry.target.getAttribute('data-item-id')
            if (itemId) {
              onActiveItemChange(itemId)
            }
          }
        })
      },
      {
        rootMargin: '-100px 0px -80% 0px',
        threshold: 0,
      }
    )

    sectionRefs.current.forEach((ref) => {
      if (ref) observer.observe(ref)
    })

    return () => {
      sectionRefs.current.forEach((ref) => {
        if (ref) observer.unobserve(ref)
      })
    }
  }, [onActiveItemChange])

  const setRef = (itemId: string, element: HTMLElement | null) => {
    if (element) {
      sectionRefs.current.set(itemId, element)
    } else {
      sectionRefs.current.delete(itemId)
    }
  }

  return (
    <div className="space-y-12">
      {categories.map((category) => (
        <div
          key={category.id}
          className="nofx-glass p-8 rounded-xl border border-white/5"
        >
          <div className="flex items-center gap-3 mb-6 pb-3 border-b border-white/10">
            <category.icon className="w-7 h-7 text-nofx-gold" />
            <h2 className="text-2xl font-bold text-nofx-text-main">
              {t(category.titleKey, language)}
            </h2>
          </div>

          <div className="space-y-8">
            {category.items.map((item) => (
              <section
                key={item.id}
                id={item.id}
                data-item-id={item.id}
                ref={(el) => setRef(item.id, el)}
                className="scroll-mt-24"
              >
                <h3 className="text-xl font-semibold mb-3 text-nofx-text-main">
                  {t(item.questionKey, language)}
                </h3>

                <div className="prose prose-invert max-w-none text-nofx-text-muted leading-relaxed">
                  {item.id === 'github-projects-tasks' ? (
                    <ContributionTasks language={language} />
                  ) : item.id === 'contribute-pr-guidelines' ? (
                    <PRGuidelines language={language} />
                  ) : (
                    <p className="text-base">{t(item.answerKey, language)}</p>
                  )}
                </div>

                <div className="mt-6 h-px bg-white/5" />
              </section>
            ))}
          </div>
        </div>
      ))}
    </div>
  )
}

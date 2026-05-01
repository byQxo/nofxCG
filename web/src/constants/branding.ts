const REPO_OWNER = 'byQxo'
const REPO_NAME = 'nofxCG'
const DEFAULT_BRANCH = 'main'

const REPO_BASE = `https://github.com/${REPO_OWNER}/${REPO_NAME}`
const BLOB_BASE = `${REPO_BASE}/blob/${DEFAULT_BRANCH}`
const RAW_BASE = `https://raw.githubusercontent.com/${REPO_OWNER}/${REPO_NAME}/${DEFAULT_BRANCH}`

export const REPO_COORDINATES = {
  owner: REPO_OWNER,
  repo: REPO_NAME,
  defaultBranch: DEFAULT_BRANCH,
} as const

export const PROJECT_LINKS = {
  repo: REPO_BASE,
  issues: `${REPO_BASE}/issues`,
  pulls: `${REPO_BASE}/pulls`,
  forks: `${REPO_BASE}/fork`,
  contributors: `${REPO_BASE}/graphs/contributors`,
  license: `${BLOB_BASE}/LICENSE`,
  disclaimer: `${BLOB_BASE}/DISCLAIMER.md`,
  readmeEn: `${BLOB_BASE}/README.md`,
  readmeZh: `${BLOB_BASE}/README_中文.md`,
  contributing: `${BLOB_BASE}/CONTRIBUTING.md`,
  prGuide: `${BLOB_BASE}/.github/PR_TITLE_GUIDE.md`,
  bountyLabel: `${REPO_BASE}/labels/bounty`,
  bountyClaim: `${BLOB_BASE}/.github/ISSUE_TEMPLATE/bounty_claim.md`,
  installScript: `${RAW_BASE}/install.sh`,
} as const

export const OFFICIAL_LINKS = {
  github: PROJECT_LINKS.repo,
  twitter: 'https://x.com/nofx_official',
  telegram: 'https://t.me/nofx_dev_community',
} as const

export const DOC_LINKS = {
  readmeEn: PROJECT_LINKS.readmeEn,
  readmeZh: PROJECT_LINKS.readmeZh,
  license: PROJECT_LINKS.license,
  disclaimer: PROJECT_LINKS.disclaimer,
  contributing: PROJECT_LINKS.contributing,
  prGuide: PROJECT_LINKS.prGuide,
} as const

export const UPSTREAM_ATTRIBUTION =
  'nofxCG is an independent derivative of upstream NOFX.'

export const LICENSE_NOTICE =
  'Distributed under AGPL-3.0. If you provide network access, you must provide the corresponding source code.'

export const BRAND_INFO = {
  name: 'nofxCG',
  displayName: 'nofxCG',
  attributionName: 'nofxCG (fork of NOFX)',
  tagline: 'Open-source self-hosted trading fork',
  version: '1.0.0',
} as const

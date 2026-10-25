import type { Locale } from './i18n';

// Chrome shared by every page that is not a component page. The component pages
// keep their own dictionary in i18n.ts; both are typed so a missing translation
// is a compile error and a test failure rather than mixed-language prose.
const english = {
  skip: 'Skip to content', home: 'Viva home',
  navDocs: 'Docs', navComponents: 'Components', navApi: 'API',
  primaryNavigation: 'Primary', language: 'Documentation language',
  repository: 'Viva on GitHub',
  onThisPage: 'On this page',
  groupStart: 'Start', groupConcepts: 'Concepts', groupGuides: 'Guides', groupDistribute: 'Build and ship', groupMore: 'More',
  docsTitle: 'Documentation', docsSummary: 'Guides for building desktop applications with Viva: the programming model, components, packaging, and what each platform supports today.',
  apiTitle: 'API reference', apiSummary: 'Every exported type, function and constant of the Viva packages, generated from the Go source and its doc comments.',
  apiPackages: 'Packages', apiType: 'Type', apiPackage: 'Package', apiTypes: 'Types', apiFunctions: 'Functions', apiConstants: 'Constants', apiVariables: 'Variables',
  apiConstructors: 'Constructors', apiMethods: 'Methods', apiOverview: 'Overview', apiImport: 'Import path',
  apiEnglishOnly: 'API documentation is generated from the Go source and is in English.',
  notFoundGuide: 'This guide has no translation yet.',
};

const chinese: typeof english = {
  skip: '跳转到正文', home: 'Viva 首页',
  navDocs: '文档', navComponents: '组件', navApi: 'API',
  primaryNavigation: '主导航', language: '文档语言',
  repository: 'Viva 的 GitHub 仓库',
  onThisPage: '本页内容',
  groupStart: '开始', groupConcepts: '概念', groupGuides: '指南', groupDistribute: '构建与发布', groupMore: '更多',
  docsTitle: '文档', docsSummary: '用 Viva 构建桌面应用的指南：编程模型、组件、打包，以及各平台目前实际支持什么。',
  apiTitle: 'API 参考', apiSummary: 'Viva 各个包导出的全部类型、函数与常量，由 Go 源码及其文档注释生成。',
  apiPackages: '包', apiType: '类型', apiPackage: '包', apiTypes: '类型', apiFunctions: '函数', apiConstants: '常量', apiVariables: '变量',
  apiConstructors: '构造函数', apiMethods: '方法', apiOverview: '概览', apiImport: '导入路径',
  apiEnglishOnly: 'API 文档由 Go 源码生成，内容为英文。',
  notFoundGuide: '这篇指南暂无翻译。',
};

export const site = { zh: chinese, en: english } as const;
export function siteText(locale: Locale): typeof english {
  return site[locale];
}

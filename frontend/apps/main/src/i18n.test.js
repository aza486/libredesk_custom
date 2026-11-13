import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import { describe, expect, test } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { initI18n } from './i18n'
import { createColumns } from './features/admin/status/dataTableColumns'
import MacroActionsPreview from './features/conversation/MacroActionsPreview.vue'
import { TooltipProvider } from '@shared-ui/components/ui/tooltip'
import { CONVERSATION_DEFAULT_STATUSES } from './constants/conversation'
import { OPERATOR, operatorLabel } from './constants/filterConfig'

const catalog = (locale) => JSON.parse(readFileSync(new URL(`../../../../i18n/${locale}.json`, import.meta.url), 'utf8'))
const de = catalog('de-DE')
const en = catalog('en-US')
const require = createRequire(import.meta.url)
const { baseCompile } = createRequire(require.resolve('vue-i18n'))('@intlify/message-compiler')
const i18n = () => initI18n({
  legacy: false,
  locale: 'de-DE',
  fallbackLocale: 'en-US',
  missingWarn: false,
  fallbackWarn: false,
  messages: { 'de-DE': de, 'en-US': en }
})

function parameters(ast, result = new Set()) {
  if (!ast || typeof ast !== 'object') return result
  if (ast.type === 4) result.add(ast.key)
  if (ast.type === 5) result.add(String(ast.index))
  for (const value of Object.values(ast)) {
    if (Array.isArray(value)) value.forEach((node) => parameters(node, result))
    else if (value && typeof value === 'object') parameters(value, result)
  }
  return result
}

// Existing catalog drift is documented in i18n-audit/README.md, section C.
// These old placeholders remain intact under the user's preservation rule.
const legacyParameters = new Set([
  'ai.apiKeyNotSet', 'report.csat.cardTitle', 'report.messages.cardTitle',
  'report.tags.cardTitle', 'validation.invalid',
  ...['add', 'all', 'apply', 'block', 'create', 'delete', 'edit', 'errorAlreadyExists',
    'import', 'new', 'no', 'notAllowed', 'notFound', 'reset', 'send', 'unblock',
    'update', 'uploading', 'yes'].map((key) => `globals.messages.${key}`)
])

describe('German catalog coverage and message contracts', () => {
  test('covers every English key, without empty translations or duplicate JSON keys', () => {
    expect(Object.keys(en).filter((key) => !Object.hasOwn(de, key))).toEqual([])
    expect(Object.entries(de).filter(([, value]) => typeof value !== 'string' || !value.trim())).toEqual([])
    const raw = readFileSync(new URL('../../../../i18n/de-DE.json', import.meta.url), 'utf8')
    const keys = [...raw.matchAll(/^\s*"([^"\n]+)":/gm)].map((match) => match[1])
    expect(new Set(keys).size).toBe(keys.length)
  })

  test('all German messages compile and all current parameters and plural branches are preserved', () => {
    const errors = []
    for (const [key, value] of Object.entries(de)) {
      const ast = baseCompile(value, { onError: (error) => errors.push(`${key}: ${error.message}`) }).ast
      if (!Object.hasOwn(en, key) || legacyParameters.has(key)) continue
      const source = baseCompile(en[key], { onError: () => {} }).ast
      expect([...parameters(ast)].sort(), key).toEqual([...parameters(source)].sort())
      expect(ast.body.type, key).toBe(source.body.type)
      if (source.body.cases) expect(ast.body.cases.length, key).toBe(source.body.cases.length)
    }
    expect(errors).toEqual([])
  })

  test('literal calls and centrally stored translation keys resolve in German', () => {
    const sources = import.meta.glob([
      './**/*.{vue,js}', '../../widget/src/**/*.{vue,js}',
      '../../../shared-ui/**/*.{vue,js}', '../../../src/**/*.{vue,js}', '!**/*.{test,spec}.js'
    ], { query: '?raw', import: 'default', eager: true })
    const missing = []
    for (const [file, source] of Object.entries(sources)) {
      for (const match of source.matchAll(/(?:\b(?:t|\$t)\(\s*|\b(?:titleKey|labelKey|allLabelKey):\s*)['"]([^'"]+)['"]/g)) {
        // Custom navigation captions also pass through t(); they are deliberately unchanged.
        if (['Service-Mails', 'KI-Lernverwaltung'].includes(match[1])) continue
        if (!Object.hasOwn(de, match[1])) missing.push(`${file}: ${match[1]}`)
      }
    }
    expect(missing).toEqual([])
  })

  test('shows German text, plural forms and intact substitutions instead of English fallback', () => {
    const { t } = i18n().global
    expect(t('globals.messages.connectionFailedRefresh')).toBe('Verbindung fehlgeschlagen. Bitte lade die Seite neu.')
    expect(t('conversation.snoozedUntil', { time: '10:30' })).toBe('Schlummernd bis 10:30')
    expect(t('conversation.bulkActions.selected', { count: 0 })).toBe('Keine Konversationen ausgewählt')
    expect(t('conversation.bulkActions.selected', { count: 1 })).toBe('1 ausgewählt')
    expect(t('conversation.bulkActions.selected', { count: 7 })).toBe('7 ausgewählt')
    expect(t('validation.invalidFromAddress')).toContain('mail@example.com')
    expect(t('admin.inbox.accessMode.description')).toContain('Konversationen')
    expect(t('admin.inbox.owner.description')).toContain('Konversationen')
    expect(t('admin.inbox.fromNameTemplate.variables')).toContain('{{ .Agent.FirstName }}')
    expect(t('audit.missingKey')).toBe('audit.missingKey')
  })

  test('keeps the configured English fallback for genuinely missing locale messages', () => {
    const instance = initI18n({ legacy: false, locale: 'de-DE', fallbackLocale: 'en-US',
      missingWarn: false, fallbackWarn: false,
      messages: { 'de-DE': {}, 'en-US': { 'audit.fallback': 'English fallback' } } })
    expect(instance.global.t('audit.fallback')).toBe('English fallback')
  })

  test('translates the existing operator mapping while keeping filter identities intact', () => {
    const filter = { field: 'tags', operator: OPERATOR.CONTAINS, value: ['Open Projects'] }
    expect(operatorLabel(filter.operator, i18n().global.t)).toBe('enthält einen der Werte')
    expect(operatorLabel(OPERATOR.NOT_CONTAINS, i18n().global.t)).toBe('enthält keinen der Werte')
    expect(filter).toEqual({ field: 'tags', operator: 'contains', value: ['Open Projects'] })
  })
})

describe('status display and protected values', () => {
  test('translates the existing default-status mapping and keeps custom names and row identities', async () => {
    const instance = i18n()
    const columns = createColumns(instance.global.t)
    const rows = ['Open', 'Closed', 'Snoozed', 'Resolved', 'Open Projects', 'Answered', 'Spam']
      .map((name, id) => ({ id, name, category: 'open' }))
    const before = structuredClone(rows)
    const app = createSSRApp({ render: () => h('div', rows.map((original) => columns[0].cell({
      row: { original, getValue: (key) => original[key] }
    }))) })
    app.use(instance)
    const html = await renderToString(app)
    for (const label of ['Offen', 'Geschlossen', 'Schlummernd', de['globals.terms.resolved'], 'Open Projects', 'Answered', 'Spam']) {
      expect(html).toContain(`>${label}</div>`)
    }
    expect(rows).toEqual(before)
    expect(CONVERSATION_DEFAULT_STATUSES).toEqual({ OPEN: 'Open', SNOOZED: 'Snoozed', RESOLVED: 'Resolved', CLOSED: 'Closed' })
    expect(rows.filter((row) => row.name === CONVERSATION_DEFAULT_STATUSES.OPEN).map((row) => row.id)).toEqual([0])
  })

  test.each([['open', 'Offen'], ['waiting', 'Wartend'], ['resolved', de['globals.terms.resolved']]])('category %s renders as %s without modifying filter values', (value, label) => {
    const original = { category: value }
    const cell = createColumns(i18n().global.t)[1].cell({ row: { original, getValue: (key) => original[key] } })
    expect(cell.children).toBe(label)
    expect(original.category).toBe(value)
  })

  test('keeps an explicitly hardcoded accessibility label in the actual component', async () => {
    const app = createSSRApp({ render: () => h(TooltipProvider, null, {
      default: () => h(MacroActionsPreview, {
        actions: [{ type: 'set_status', value: ['Open Projects'] }], onRemove: () => {}
      })
    }) })
    app.use(i18n())
    const html = await renderToString(app)
    expect(html).toContain('Remove action')
    expect(html).toContain('Open Projects')
  })
})

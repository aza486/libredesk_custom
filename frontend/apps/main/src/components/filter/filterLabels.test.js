// @vitest-environment jsdom
import { expect, test } from 'vitest'
import { createApp, h, nextTick } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createI18n } from 'vue-i18n'
import de from '../../../../../../i18n/de-DE.json'
import en from '../../../../../../i18n/en-US.json'
import { FIELD_OPERATORS, FIELD_TYPE, OPERATOR, OPERATOR_LABEL_KEYS, operatorLabel } from '../../constants/filterConfig'
import { normalizeToTwoLevel, serializeFilterTree } from './filterTree'
import FilterRow from './FilterRow.vue'
import FilterBuilder from './FilterBuilder.vue'
import ConnectorToggle from './ConnectorToggle.vue'

const instance = () => createI18n({ legacy: false, locale: 'de-DE', fallbackLocale: 'en-US', messages: { 'de-DE': de, 'en-US': en } })

test('every supported operator has German and English keys, including automation starts with', () => {
  const i18n = instance()
  const expected = ['Ist gleich', 'Ist nicht gleich', 'Ist gesetzt', 'Ist nicht gesetzt', 'Enthält', 'Enthält nicht', 'Größer als', 'Kleiner als', 'Zwischen', 'Beginnt mit']
  expect(Object.values(OPERATOR).map(op => operatorLabel(op, i18n.global.t, false))).toEqual(expected)
  for (const op of new Set(Object.values(FIELD_OPERATORS).flat())) {
    expect(de[OPERATOR_LABEL_KEYS[op]]).toBeTruthy()
    expect(en[OPERATOR_LABEL_KEYS[op]]).toBeTruthy()
  }
  expect(operatorLabel(OPERATOR.CONTAINS, i18n.global.t)).toBe('enthält einen der Werte')
  i18n.global.locale.value = 'en-US'
  expect(Object.values(OPERATOR).map(op => operatorLabel(op, i18n.global.t, false))).toEqual(Object.values(OPERATOR))
})

test.each([FilterRow, FilterBuilder])('filter controls translate selected operators on language changes without changing saved values', async (Component) => {
  const i18n = instance()
  const leaf = { model: 'conversations', field: 'subject', operator: 'not equals', value: 'custom subject' }
  const original = JSON.stringify(leaf)
  const app = createApp({ render: () => h(Component, {
    fields: [{ field: 'subject', label: 'Custom subject label', type: FIELD_TYPE.TEXT, operators: FIELD_OPERATORS.TEXT }],
    modelValue: Component === FilterRow ? leaf : [leaf]
  }) })
  app.use(i18n)
  const host = document.createElement('div')
  document.body.appendChild(host)
  app.mount(host)
  await nextTick()
  expect(host.textContent).toContain('Ist nicht gleich')
  expect(host.textContent).toContain('Custom subject label')
  i18n.global.locale.value = 'en-US'
  await nextTick()
  expect(host.textContent).toContain('not equals')
  expect(JSON.stringify(leaf)).toBe(original)
  app.unmount()
  host.remove()
})

test('group connectors translate while preserving AND/OR and legacy custom-folder structures', async () => {
  const i18n = instance()
  for (const [logic, label] of [['AND', 'Und'], ['OR', 'Oder']]) {
    const app = createApp({ render: () => h(ConnectorToggle, { modelValue: logic }) })
    app.use(i18n)
    expect((await renderToString(app)).toLowerCase()).toContain(label.toLowerCase())
  }
  const saved = [{ model: 'conversations', field: 'subject', operator: 'contains', value: 'Support' }, { model: 'conversations', field: 'priority_id', operator: 'equals', value: '2' }]
  const before = JSON.stringify(saved)
  const normalized = normalizeToTwoLevel(saved)
  for (const locale of ['de-DE', 'en-US']) {
    i18n.global.locale.value = locale
    for (const leaf of saved) operatorLabel(leaf.operator, i18n.global.t, false)
    expect(JSON.stringify(saved)).toBe(before)
    expect(serializeFilterTree(normalized).rules[0].rules.map(({ operator }) => operator)).toEqual(['contains', 'equals'])
  }
})

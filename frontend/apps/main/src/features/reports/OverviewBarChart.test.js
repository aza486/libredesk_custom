import { expect, test, vi } from 'vitest'
import { createSSRApp, h } from 'vue'
import { renderToString } from 'vue/server-renderer'
import { createI18n } from 'vue-i18n'
import de from '../../../../../../i18n/de-DE.json'
import OverviewBarChart from './OverviewBarChart.vue'

const state = vi.hoisted(() => ({ props: null }))
vi.mock('@shared-ui/components/ui/chart-bar', () => ({ BarChart: {
  props: ['data', 'categories'], setup(props) { state.props = props; return () => null }
} }))
async function render(data) {
  const app = createSSRApp({ render: () => h(OverviewBarChart, { data }) })
  app.use(createI18n({ legacy: false, locale: 'de-DE', messages: { 'de-DE': de } }))
  await renderToString(app)
}
test('chart legend/tooltip series translate through a presentation copy, keeping API rows intact', async () => {
  const data = Object.freeze([Object.freeze({ status: 'Open', Low: 1, Medium: 2, High: 3, Custom: 9 })])
  await render(data)
  expect(state.props.categories).toEqual(['Niedrig', 'Mittel', 'Hoch'])
  expect(state.props.data).toEqual([{ status: 'Open', Niedrig: 1, Mittel: 2, Hoch: 3, Custom: 9 }])
  expect(data[0]).toEqual({ status: 'Open', Low: 1, Medium: 2, High: 3, Custom: 9 })
})
test('custom column collisions keep their values without overwriting data', async () => {
  await render([{ status: 'Open', Low: 1, Medium: 2, High: 3, Hoch: 9 }])
  expect(state.props.categories).toEqual(['Niedrig', 'Mittel', 'High'])
  expect(state.props.data[0]).toMatchObject({ High: 3, Hoch: 9 })
})

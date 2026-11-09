import { describe, expect, test } from 'vitest'
import { createI18n } from 'vue-i18n'
import de from '../../../../../i18n/de-DE.json'
import en from '../../../../../i18n/en-US.json'
import { conversationActivityLabel } from './conversation-activity'
import { conversationStatusLabel } from './conversation-status'

const translator = (locale = 'de-DE') => createI18n({
  legacy: false, locale, fallbackLocale: 'en-US', missingWarn: false, fallbackWarn: false,
  messages: { 'de-DE': de, 'en-US': en }
}).global.t
const t = translator()
const activity = (content, type) => ({
  type: 'activity', sender_type: 'agent', private: true, content_type: 'text', content,
  ...(type ? { meta: { activity_type: type } } : {})
})

const cases = [
  ['assigned_user_change', 'Assigned to Alex Müller by Daniel', 'Daniel hat die Konversation Alex Müller zugewiesen'],
  ['assigned_team_change', 'Assigned to Rekla / Support team by Daniel', 'Daniel hat die Konversation dem Team Rekla / Support zugewiesen'],
  ['self_assign', 'Zoë Smith self-assigned this conversation', 'Zoë Smith hat sich die Konversation selbst zugewiesen'],
  ['priority_change', 'Daniel set priority to High', 'Daniel hat die Priorität auf Hoch gesetzt'],
  ['status_change', 'Daniel marked the conversation as Closed', 'Daniel hat die Konversation als Geschlossen markiert'],
  ['status_change', 'Daniel marked the conversation as Resolved', `Daniel hat die Konversation als ${de['globals.terms.resolved']} markiert`],
  ['status_change', 'Daniel marked the conversation as Open', 'Daniel hat die Konversation als Offen markiert'],
  ['status_change', 'Daniel marked the conversation as Snoozed', 'Daniel hat die Konversation als Schlummernd markiert'],
  ['status_change', 'Daniel marked the conversation as Open Projects', 'Daniel hat die Konversation als Open Projects markiert'],
  ['tag_added', 'Daniel added tag Rekla', 'Daniel hat das Tag Rekla hinzugefügt'],
  ['tag_removed', 'Özlem Yılmaz removed tag Open Projects', 'Özlem Yılmaz hat das Tag Open Projects entfernt'],
  ['sla_set', 'Daniel set Premium 24h SLA policy', 'Daniel hat die SLA-Richtlinie Premium 24h festgelegt'],
  ['participant_added', 'Anne-Marie de Vries joined the conversation', 'Anne-Marie de Vries ist der Konversation beigetreten']
]

describe('exact status display names', () => {
  test.each([['Open', 'Offen'], ['Closed', 'Geschlossen'], ['Snoozed', 'Schlummernd'], ['Resolved', de['globals.terms.resolved']]])('%s renders as %s', (name, expected) => {
    expect(conversationStatusLabel(name, t)).toBe(expected)
    expect(conversationStatusLabel(name, translator('en-US'))).toBe(name)
  })
  test.each(['Open Projects', 'Answered', 'Spam', 'open', 'resolved', 'Closed ', 'toString', '__proto__', 'constructor'])('custom or noncanonical name %s stays intact', (name) => {
    expect(conversationStatusLabel(name, t)).toBe(name)
  })
  test('missing translator/catalog keeps the API name', () => {
    expect(conversationStatusLabel('Closed')).toBe('Closed')
    expect(conversationStatusLabel('Closed', (key) => key)).toBe('Closed')
  })
})

describe('known system activities', () => {
  test.each(cases)('%s: %s', (type, content, expected) => {
    const message = activity(content, type)
    const snapshot = structuredClone(message)
    expect(conversationActivityLabel(message, t)).toBe(expected)
    expect(conversationActivityLabel(message, translator('en-US'))).toBe(content)
    expect(message).toEqual(snapshot)
  })
  test.each(cases.filter(([type]) => type !== 'assigned_team_change'))('unambiguous historical %s: %s', (_type, content, expected) => {
    expect(conversationActivityLabel(activity(content), t)).toBe(expected)
    expect(conversationActivityLabel({ ...activity(content), meta: null }, t)).toBe(expected)
  })
  test('WS preview shape is accepted only for dedicated system activities', () => {
    const message = { type: 'activity', sender_type: 'agent', content: 'Daniel added tag Rekla' }
    expect(conversationActivityLabel(message, t)).toBe('Daniel hat das Tag Rekla hinzugefügt')
    expect(conversationActivityLabel({ ...message, type: 'outgoing' }, t)).toBe(message.content)
  })
  test('the type disambiguates user names ending in team', () => {
    const content = 'Assigned to Support team by Daniel'
    expect(conversationActivityLabel(activity(content, 'assigned_user_change'), t)).toBe('Daniel hat die Konversation Support team zugewiesen')
    expect(conversationActivityLabel(activity(content, 'assigned_team_change'), t)).toBe('Daniel hat die Konversation dem Team Support zugewiesen')
    expect(conversationActivityLabel(activity(content), t)).toBe(content)
  })
  test('dynamic values are substituted literally, including markup and interpolation syntax', () => {
    expect(conversationActivityLabel(activity("Zoë {'@'} added tag <b>Open {count}</b>", 'tag_added'), t))
      .toBe("Zoë {'@'} hat das Tag <b>Open {count}</b> hinzugefügt")
  })
  test('a known type preserves a tag containing a different event phrase', () => {
    const content = 'Daniel added tag Foo joined the conversation'
    expect(conversationActivityLabel(activity(content, 'tag_added'), t)).toBe('Daniel hat das Tag Foo joined the conversation hinzugefügt')
    expect(conversationActivityLabel(activity(content), t)).toBe(content)
  })
})

describe('fail-closed original-text fallback', () => {
  test.each([
    'Custom system message', 'Daniel added tag ', ' added tag Rekla',
    'Daniel added tag Rekla\n', 'Daniel added tag Rekla\nInjected',
    ' Daniel added tag Rekla', 'Daniel added tag  Rekla',
    'Daniel added tag one added tag two', 'Assigned to Alex by Sales by Daniel',
    'Daniel set Gold set Premium SLA policy', 'Daniel set priority to Gold SLA policy',
    'Assigned to Support team by Daniel', 'Daniel hat das Tag Rekla hinzugefügt',
    'x'.repeat(8193)
  ])('leaves unsafe/unknown historical format unchanged: %s', (content) => {
    expect(conversationActivityLabel(activity(content), t)).toBe(content)
  })
  test.each([
    { type: 'incoming' }, { type: 'outgoing' }, { type: 'note' },
    { sender_type: 'contact' }, { sender_type: undefined }, { private: false },
    { content_type: 'html' }, { meta: [] }, { meta: '{"activity_type":"tag_added"}' },
    { meta: { activity_type: null } }, { meta: { activity_type: 'unknown' } },
    { meta: { activity_type: 'toString' } }, { meta: { activity_type: 'status_change' } }
  ])('keeps non-system messages or conflicting metadata unchanged: %j', (patch) => {
    const message = { ...activity('Daniel added tag Rekla', 'tag_added'), ...patch }
    expect(conversationActivityLabel(message, t)).toBe(message.content)
  })
  test.each(cases)('never translates ordinary incoming/outgoing notes containing %s', (type, content) => {
    for (const messageType of ['incoming', 'outgoing']) {
      for (const privateNote of [true, false]) {
        const message = { ...activity(content, type), type: messageType, private: privateNote }
        expect(conversationActivityLabel(message, t)).toBe(content)
      }
    }
  })
  test('missing message and missing translation do not throw or show keys', () => {
    expect(conversationActivityLabel(undefined, t)).toBeUndefined()
    const message = activity('Daniel added tag Rekla', 'tag_added')
    expect(conversationActivityLabel(message, (key) => key)).toBe(message.content)
    expect(conversationActivityLabel(message)).toBe(message.content)
  })
})

/* eslint-env node */
import { readFileSync } from 'node:fs'
import { expect, test } from 'vitest'

const read = (name) => JSON.parse(readFileSync(new URL(`../../../../../../${name}`, import.meta.url), 'utf8'))
const original = read('E-Mail Anonymisierung - Security Link Check V2 Fix(1).json')
const guarded = read('workflows/E-Mail Anonymisierung - Service Mail Guard.json')
const nodes = Object.fromEntries(guarded.nodes.map((node) => [node.name, node]))
const destinations = (name, branch = 0) => guarded.connections[name]?.main?.[branch]?.map((edge) => edge.node) || []

test('preserves original node IDs, credentials, webhook settings and unaffected nodes', () => {
  expect(guarded.active).toBe(false)
  expect(guarded.nodes).toHaveLength(original.nodes.length)
  for (const node of original.nodes) {
    expect(nodes[node.name].id).toBe(node.id)
    expect(nodes[node.name].credentials).toEqual(node.credentials)
    if (node.name !== 'Execute a SQL query2') expect(nodes[node.name]).toEqual(node)
  }
  const names = new Set(guarded.nodes.map((node) => node.name))
  for (const [source, outputs] of Object.entries(guarded.connections)) {
    expect(names.has(source)).toBe(true)
    for (const branches of Object.values(outputs)) for (const edges of branches) for (const edge of edges) expect(names.has(edge.node)).toBe(true)
  }
})

test('checks the existing LibreDesk registry with a bound conversation UUID before every branch', () => {
  expect(destinations('If')).toEqual(['Execute a SQL query2', 'Merge6'])
  expect(destinations('If4')).toEqual(['Security - Conversation created?'])
  expect(destinations('If4', 1)).toEqual([])
  expect(destinations('Switch', 3)).toEqual(['Execute a SQL query3', 'Merge7'])
  const query = nodes['Execute a SQL query2']
  expect(query.credentials).toEqual(original.nodes.find((n) => n.name === query.name).credentials)
  expect(query.parameters.query).toContain('service_email_addresses')
  expect(query.parameters.query).toContain('LOWER(BTRIM(s.address,')
  expect(query.parameters.query).toContain('LOWER(BTRIM(u.email,')
  expect(query.parameters.query).toContain("U&' ")
  expect(query.parameters.query).toContain('c.uuid=$1::uuid')
  expect(query.parameters.query).not.toContain('{{')
  expect(query.parameters.options.queryReplacement).toContain('Normalize Event')
  expect(query.onError).toBe('stopWorkflow')
  const condition = nodes.If4.parameters.conditions.conditions[0]
  expect(condition.rightValue).toBe(false)
  expect(condition.operator.type).toBe('boolean')
  expect(nodes.If4.parameters.conditions.options.typeValidation).toBe('strict')
  // Remove the guard's allow edge: no HTTP, SQL processing or LLM is reachable.
  const visited = new Set()
  function visit(name) {
    if (visited.has(name)) return
    visited.add(name)
    if (name === 'If4') return
    for (const branches of Object.values(guarded.connections[name] || {})) for (const edges of branches) for (const edge of edges) visit(edge.node)
  }
  visit('Webhook')
  expect([...visited].sort()).toEqual(['Webhook', 'Normalize Event', 'If', 'Execute a SQL query2', 'Merge6', 'If4'].sort())
})

test('normalization preserves public/private and excludes employee notes and outgoing events', () => {
  const normalize = new Function('$json', nodes['Normalize Event'].parameters.jsCode)
  for (const mode of ['public', 'personal']) {
    const result = normalize({ body: { event: 'message.created', payload: { uuid: 'm', conversation_uuid: 'c', type: 'incoming', sender_type: 'contact', private: false, inbox_access_mode: mode } } })[0].json
    expect(result.is_relevant_customer_event).toBe(true)
    expect(result.mail_type).toBe(mode === 'public' ? 'customer' : 'private_mail')
    expect(result.conversation_uuid).toBe('c')
  }
  for (const payload of [{ type: 'outgoing', sender_type: 'agent' }, { type: 'incoming', sender_type: 'contact', private: true }]) expect(normalize({ body: { event: 'message.created', payload } })[0].json.is_relevant_customer_event).toBe(false)
  expect(() => normalize({ body: { event: 'conversation.created', payload: {} } })).toThrow()
})

test('reply and initial-message extraction still select the original incoming message', () => {
  const normalized = { event_type: 'message.created', conversation_uuid: 'c', message_uuid: 'm', body: { payload: { uuid: 'm', id: 1, conversation_id: 2, type: 'incoming', sender_type: 'contact', private: false, text_content: 'Original customer', author: { email: 'sender@example.com' } } } }
  const lookup = (name) => ({ first: () => ({ json: name === 'Normalize Event' ? normalized : {} }) })
  const run = new Function('$input', '$', nodes.Filter.parameters.jsCode)
  const result = run({ first: () => ({ json: { data: { results: [{ uuid: 'note', type: 'outgoing', sender_type: 'agent', private: true, text_content: 'Security note' }] } } }) }, lookup)[0].json
  expect(result.message_uuid).toBe('m')
  expect(result.raw_text).toBe('Original customer')
})

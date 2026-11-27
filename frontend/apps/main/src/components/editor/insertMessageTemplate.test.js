// @vitest-environment jsdom
import { afterEach, expect, test } from 'vitest'
import { Editor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import examples from '../../../../../../internal/messagetemplate/testdata/examples.json'
import { captureTemplatePosition, insertMessageTemplate } from './insertMessageTemplate'
let editor
const create = (content = '') =>
  (editor = new Editor({ extensions: [StarterKit, Underline, Link], content }))
afterEach(() => editor?.destroy())
const sample =
  '<p>Sehr geehrte/r [Name],</p><p>Vielen Dank.<br>Gelangensbestätigung</p><ul><li><strong>Einfache Abrechnung</strong> – <em>monatlich</em> <u>Details</u></li></ul><ol><li>Erster Punkt</li></ol><p><a href="https://www.pixelfoto-express.de/scan-und-film/filmentwicklung">Informationen und Preise</a></p>'
test('inserts formatted paragraphs, links and manual placeholders into an empty editor', () => {
  create()
  expect(insertMessageTemplate(editor, sample, null)).toBe(true)
  const html = editor.getHTML()
  for (const part of [
    '[Name]',
    '<br>',
    '<strong>',
    '<em>',
    '<u>',
    '<ul>',
    '<ol>',
    'https://www.pixelfoto-express.de/scan-und-film/filmentwicklung'
  ])
    expect(html).toContain(part)
  expect(editor.getText()).toContain('Informationen und Preise')
})
test('preserves existing draft at the cursor, including selected draft text', () => {
  create('<p>Vorher Nachher</p>')
  editor.commands.setTextSelection({ from: 1, to: 7 })
  const position = captureTemplatePosition(editor)
  expect(insertMessageTemplate(editor, '<p>Vorlage</p>', position)).toBe(true)
  expect(editor.getText()).toMatch(/Vorher[\s\S]*Vorlage[\s\S]*Nachher/)
})
test('appends with missing or invalid selection and leaves the stored source unchanged', () => {
  create('<p>KI-Antwortentwurf</p>')
  insertMessageTemplate(editor, sample, 99999)
  expect(editor.getText()).toMatch(/^KI-Antwortentwurf[\s\S]*Sehr geehrte/)
  editor.commands.insertContent('Manuelle Änderung')
  expect(sample).not.toContain('Manuelle Änderung')
  expect(editor.getHTML()).not.toContain('outgoing_signature')
})
test('does not modify a disabled editor', () => {
  create('<p>Entwurf</p>')
  editor.setEditable(false)
  expect(insertMessageTemplate(editor, sample, null)).toBe(false)
  expect(editor.getHTML()).toBe('<p>Entwurf</p>')
})

test('preserves both full development examples, long lists and all paragraphs', () => {
  create('<p>Erhaltener Entwurf</p>')
  for (const example of examples) insertMessageTemplate(editor, example.content, null)
  const html = editor.getHTML()
  expect(editor.getText()).toContain('Vielen Dank für Ihre Unterstützung und Ihr Verständnis.')
  expect(editor.getText()).toContain(
    'Wir sind auf eine neue und schöne Zusammenarbeit mit Ihnen sehr gespannt.'
  )
  expect(editor.getText()).toContain('[Name]')
  expect((html.match(/<li>/g) || []).length).toBe(7)
  expect((html.match(/<strong>/g) || []).length).toBe(7)
  expect(html).toContain('href="https://www.pixelfoto-express.de/scan-und-film/filmentwicklung"')
  expect((html.match(/<p>/g) || []).length).toBeGreaterThan(18)
})

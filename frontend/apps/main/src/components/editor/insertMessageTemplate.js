// Keep the insertion anchor across menu focus changes. A selection inserts after
// its end so no selected draft text is silently replaced.
export function captureTemplatePosition(editor) {
  if (!editor || editor.isDestroyed) return null
  return editor.state.selection.to
}
export function insertMessageTemplate(editor, content, position) {
  if (!editor || editor.isDestroyed || !editor.isEditable) return false
  const end = editor.state.doc.content.size
  const target = Number.isInteger(position) && position >= 1 && position < end ? position : end - 1
  return editor.chain().focus().insertContentAt(target, content).run()
}

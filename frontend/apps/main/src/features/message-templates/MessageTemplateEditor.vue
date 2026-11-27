<script setup>
import { watch } from 'vue'
import { EditorContent, useEditor } from '@tiptap/vue-3'
import StarterKit from '@tiptap/starter-kit'
import Underline from '@tiptap/extension-underline'
import Link from '@tiptap/extension-link'
import {
  Bold,
  Italic,
  Underline as UnderlineIcon,
  List,
  ListOrdered,
  Link as LinkIcon
} from 'lucide-vue-next'
import { Button } from '@shared-ui/components/ui/button'
import EditorLinkDialog from '@main/components/editor/EditorLinkDialog.vue'
import { ref } from 'vue'
const content = defineModel({ default: '' })
const props = defineProps({ disabled: Boolean })
const linkDialog = ref(null)
const editor = useEditor({
  extensions: [
    StarterKit.configure({
      heading: false,
      codeBlock: false,
      code: false,
      blockquote: false,
      horizontalRule: false,
      strike: false
    }),
    Underline,
    Link.configure({ openOnClick: false })
  ],
  content: content.value,
  editable: !props.disabled,
  editorProps: {
    attributes: {
      class: 'outline-none min-h-48 p-4',
      role: 'textbox',
      'aria-label': 'Vorlageninhalt',
      'aria-multiline': 'true'
    }
  },
  onUpdate: ({ editor }) => {
    content.value = editor.getHTML()
  }
})
watch(content, (value) => {
  if (editor.value && value !== editor.value.getHTML())
    editor.value.commands.setContent(value, false)
})
watch(
  () => props.disabled,
  (value) => editor.value?.setEditable(!value)
)
const tools = [
  { icon: Bold, label: 'Fett', mark: 'bold', command: 'toggleBold' },
  { icon: Italic, label: 'Kursiv', mark: 'italic', command: 'toggleItalic' },
  { icon: UnderlineIcon, label: 'Unterstreichen', mark: 'underline', command: 'toggleUnderline' },
  { icon: List, label: 'Aufzählung', mark: 'bulletList', command: 'toggleBulletList' },
  {
    icon: ListOrdered,
    label: 'Nummerierte Liste',
    mark: 'orderedList',
    command: 'toggleOrderedList'
  }
]
</script>
<template>
  <div class="rounded-md border min-h-0 flex flex-col overflow-hidden">
    <div class="flex flex-wrap border-b p-1 shrink-0" role="toolbar" aria-label="Textformatierung">
      <Button
        v-for="tool in tools"
        :key="tool.mark"
        type="button"
        variant="ghost"
        size="sm"
        :title="tool.label"
        :aria-label="tool.label"
        :aria-pressed="editor?.isActive(tool.mark)"
        :disabled="disabled"
        @click="editor?.chain().focus()[tool.command]().run()"
      >
        <component :is="tool.icon" class="size-4" />
      </Button>
      <Button
        type="button"
        variant="ghost"
        size="sm"
        title="Link"
        aria-label="Link"
        :disabled="disabled"
        @click="linkDialog?.open()"
        ><LinkIcon class="size-4"
      /></Button>
    </div>
    <EditorContent :editor="editor" class="native-html overflow-y-auto min-h-0 flex-1" />
    <EditorLinkDialog ref="linkDialog" :editor="editor" :allow-button="false" />
  </div>
</template>
<style lang="scss" src="../../components/editor/editorStyles.scss"></style>

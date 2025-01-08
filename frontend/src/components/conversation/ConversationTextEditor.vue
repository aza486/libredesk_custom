<template>
  <div class="max-h-[600px] overflow-y-auto">
    <BubbleMenu :editor="editor" :tippy-options="{ duration: 100 }" v-if="editor" class="bg-white p-1 box rounded-lg">
      <div class="flex space-x-1 items-center">
        <DropdownMenu>
          <DropdownMenuTrigger>
            <Button size="sm" variant="ghost" class="flex items-center justify-center">
              <span class="flex items-center">
                <span class="text-medium">AI</span>
                <Bot size="14" class="ml-1" />
                <ChevronDown class="w-4 h-4 ml-2" />
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem v-for="prompt in aiPrompts" :key="prompt.key" @select="emitPrompt(prompt.key)">
              {{ prompt.title }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button size="sm" variant="ghost" @click="isBold = !isBold" :active="isBold" :class="{ 'bg-gray-200': isBold }">
          <Bold size="14" />
        </Button>
        <Button size="sm" variant="ghost" @click="isItalic = !isItalic" :active="isItalic"
          :class="{ 'bg-gray-200': isItalic }">
          <Italic size="14" />
        </Button>
      </div>
    </BubbleMenu>
    <EditorContent :editor="editor" />
  </div>
</template>

<script setup>
import { ref, watch, watchEffect, onUnmounted } from 'vue'
import { useEditor, EditorContent, BubbleMenu } from '@tiptap/vue-3'
import { ChevronDown, Bold, Italic, Bot } from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'

const selectedText = defineModel('selectedText', { default: '' })
const textContent = defineModel('textContent')
const htmlContent = defineModel('htmlContent')
const isBold = defineModel('isBold')
const isItalic = defineModel('isItalic')
const cursorPosition = defineModel('cursorPosition', {
  default: 0
})

const props = defineProps({
  placeholder: String,
  clearContent: Boolean,
  contentToSet: String,
  aiPrompts: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits([
  'send',
  'editorReady',
  'aiPromptSelected'
])

function emitPrompt (key) {
  emit('aiPromptSelected', key)
}

const editor = ref(
  useEditor({
    content: textContent.value,
    extensions: [
      StarterKit,
      Image.configure({
        HTMLAttributes: { class: 'inline-image' }
      }),
      Placeholder.configure({
        placeholder: () => props.placeholder
      }),
      Link
    ],
    autofocus: true,
    editorProps: { attributes: { class: 'outline-none' } },
    onSelectionUpdate: ({ editor }) => {
      selectedText.value = editor.state.doc.textBetween(
        editor.state.selection.from,
        editor.state.selection.to
      )
    },
    onUpdate: ({ editor }) => {
      htmlContent.value = editor.getHTML()
      textContent.value = editor.getText()
      cursorPosition.value = editor.state.selection.from
    },
    onCreate: ({ editor }) => {
      if (cursorPosition.value) {
        editor.commands.setTextSelection(cursorPosition.value)
      }
    },
  })
)

watchEffect(() => {
  if (editor.value) {
    emit('editorReady', editor.value)
    isBold.value = editor.value.isActive('bold')
    isItalic.value = editor.value.isActive('italic')
  }
})

watchEffect(() => {
  if (isBold.value !== editor.value?.isActive('bold')) {
    isBold.value
      ? editor.value?.chain().focus().setBold().run()
      : editor.value?.chain().focus().unsetBold().run()
  }
  if (isItalic.value !== editor.value?.isActive('italic')) {
    isItalic.value
      ? editor.value?.chain().focus().setItalic().run()
      : editor.value?.chain().focus().unsetItalic().run()
  }
})

watch(
  () => props.contentToSet,
  (newContent) => {
    if (newContent) {
      console.log('Setting content to -:', newContent)
      editor.value.commands.setContent(newContent)
      editor.value.commands.focus()
    }
  }
)

watch(cursorPosition, (newPos, oldPos) => {
  if (editor.value && newPos !== oldPos && newPos !== editor.value.state.selection.from) {
    editor.value.commands.setTextSelection(newPos)
  }
})

onUnmounted(() => {
  editor.value.destroy()
})
</script>

<style lang="scss">
// Moving placeholder to the top.
.tiptap p.is-editor-empty:first-child::before {
  content: attr(data-placeholder);
  float: left;
  color: #adb5bd;
  pointer-events: none;
  height: 0;
}

// Editor height
.ProseMirror {
  min-height: 150px !important;
  max-height: 100% !important;
  overflow-y: scroll !important;
  padding: 10px 10px;
}

.tiptap {
  a {
    color: #0066cc;
    cursor: pointer;

    &:hover {
      color: #003d7a;
    }
  }
}

br.ProseMirror-trailingBreak {
  display: none;
}
</style>
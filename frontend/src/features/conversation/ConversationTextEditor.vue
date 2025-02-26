<template>
  <div class="editor-wrapper h-full overflow-y-auto">
    <BubbleMenu
      :editor="editor"
      :tippy-options="{ duration: 100 }"
      v-if="editor"
      class="bg-white p-1 box will-change-transform"
    >
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
            <DropdownMenuItem
              v-for="prompt in aiPrompts"
              :key="prompt.key"
              @select="emitPrompt(prompt.key)"
            >
              {{ prompt.title }}
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
        <Button
          size="sm"
          variant="ghost"
          @click="isBold = !isBold"
          :active="isBold"
          :class="{ 'bg-gray-200': isBold }"
        >
          <Bold size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click="isItalic = !isItalic"
          :active="isItalic"
          :class="{ 'bg-gray-200': isItalic }"
        >
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
  DropdownMenuTrigger
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
const cursorPosition = defineModel('cursorPosition', { default: 0 })

const props = defineProps({
  placeholder: String,
  contentToSet: String,
  setInlineImage: Object,
  insertContent: String,
  clearContent: Boolean,
  aiPrompts: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['send', 'aiPromptSelected'])

const emitPrompt = (key) => emit('aiPromptSelected', key)

const getSelectionText = (from, to, doc) => doc.textBetween(from, to)

const editorConfig = {
  extensions: [
    // Lists are unstyled in tailwind, so need to add classes to them.
    StarterKit.configure({
      bulletList: {
        HTMLAttributes: {
          class: 'list-disc ml-6 my-2'
        }
      },
      orderedList: {
        HTMLAttributes: {
          class: 'list-decimal ml-6 my-2'
        }
      },
      listItem: {
        HTMLAttributes: {
          class: 'pl-1'
        }
      },
      heading: {
        HTMLAttributes: {
          class: 'text-xl font-bold mt-4 mb-2'
        }
      }
    }),
    Image.configure({ HTMLAttributes: { class: 'inline-image' } }),
    Placeholder.configure({ placeholder: () => props.placeholder }),
    Link
  ],
  autofocus: true,
  editorProps: {
    attributes: { class: 'outline-none' },
    handleKeyDown: (view, event) => {
      if (event.ctrlKey && event.key === 'Enter') {
        emit('send')
        return true
      }
    }
  }
}

const editor = ref(
  useEditor({
    ...editorConfig,
    content: htmlContent.value,
    onSelectionUpdate: ({ editor }) => {
      const { from, to } = editor.state.selection
      selectedText.value = getSelectionText(from, to, editor.state.doc)
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
    }
  })
)

watchEffect(() => {
  const editorInstance = editor.value
  if (!editorInstance) return
  isBold.value = editorInstance.isActive('bold')
  isItalic.value = editorInstance.isActive('italic')
})

watchEffect(() => {
  const editorInstance = editor.value
  if (!editorInstance) return

  if (isBold.value !== editorInstance.isActive('bold')) {
    isBold.value
      ? editorInstance.chain().focus().setBold().run()
      : editorInstance.chain().focus().unsetBold().run()
  }
  if (isItalic.value !== editorInstance.isActive('italic')) {
    isItalic.value
      ? editorInstance.chain().focus().setItalic().run()
      : editorInstance.chain().focus().unsetItalic().run()
  }
})

watch(
  () => props.contentToSet,
  (newContentData) => {
    if (!newContentData) return
    try {
      const parsedData = JSON.parse(newContentData)
      const content = parsedData.content
      if (content === '') {
        editor.value?.commands.clearContent()
      } else {
        editor.value?.commands.setContent(content, true)
      }
      editor.value?.commands.focus()
    } catch (e) {
      console.error('Error parsing content data', e)
    }
  }
)

watch(cursorPosition, (newPos, oldPos) => {
  if (editor.value && newPos !== oldPos && newPos !== editor.value.state.selection.from) {
    editor.value.commands.setTextSelection(newPos)
  }
})

watch(
  () => props.clearContent,
  () => {
    if (!props.clearContent) return
    editor.value?.commands.clearContent()
    editor.value?.commands.focus()
    // `onUpdate` is not called when clearing content, so need to reset the content here.
    htmlContent.value = ''
    textContent.value = ''
    cursorPosition.value = 0
  }
)

watch(
  () => props.setInlineImage,
  (val) => {
    if (val) {
      editor.value?.commands.setImage({
        src: val.src,
        alt: val.alt,
        title: val.title
      })
    }
  }
)

watch(
  () => props.insertContent,
  (val) => {
    if (val) editor.value?.commands.insertContent(val)
  }
)

onUnmounted(() => {
  editor.value?.destroy()
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

// Ensure the parent div has a proper height
.editor-wrapper div[aria-expanded='false'] {
  display: flex;
  flex-direction: column;
  height: 100%;
}

// Ensure the editor content has a proper height and breaks words
.tiptap.ProseMirror {
  flex: 1;
  min-height: 70px;
  overflow-y: auto;
  word-wrap: break-word !important;
  overflow-wrap: break-word !important;
  word-break: break-word;
  white-space: pre-wrap;
  max-width: 100%;
}

// Anchor tag styling
.tiptap {
  a {
    color: #0066cc;
    cursor: pointer;

    &:hover {
      color: #003d7a;
    }
  }
}
</style>

<template>
  <div class="editor-wrapper h-full overflow-y-auto">
    <BubbleMenu
      :editor="editor"
      :tippy-options="{ duration: 100 }"
      v-if="editor"
      class="bg-background p-1 box will-change-transform"
    >
      <div class="flex space-x-1 items-center">
        <DropdownMenu v-if="aiPrompts.length > 0">
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
          @click.prevent="isBold = !isBold"
          :active="isBold"
          :class="{ 'bg-gray-200 dark:bg-secondary': isBold }"
        >
          <Bold size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="isItalic = !isItalic"
          :active="isItalic"
          :class="{ 'bg-gray-200 dark:bg-secondary': isItalic }"
        >
          <Italic size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="toggleBulletList"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('bulletList') }"
        >
          <List size="14" />
        </Button>

        <Button
          size="sm"
          variant="ghost"
          @click.prevent="toggleOrderedList"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('orderedList') }"
        >
          <ListOrdered size="14" />
        </Button>
        <Button
          size="sm"
          variant="ghost"
          @click.prevent="openLinkModal"
          :class="{ 'bg-gray-200 dark:bg-secondary': editor?.isActive('link') }"
        >
          <LinkIcon size="14" />
        </Button>
        <div v-if="showLinkInput" class="flex space-x-2 p-2 bg-background border rounded">
          <Input
            v-model="linkUrl"
            type="text"
            placeholder="Enter link URL"
            class="border p-1 text-sm w-[200px]"
          />
          <Button size="sm" @click="setLink">
            <Check size="14" />
          </Button>
          <Button size="sm" @click="unsetLink">
            <X size="14" />
          </Button>
        </div>
      </div>
    </BubbleMenu>
    <EditorContent :editor="editor" class="native-html" />
  </div>
</template>

<script setup>
import { ref, watch, watchEffect, onUnmounted, computed } from 'vue'
import { useEditor, EditorContent, BubbleMenu } from '@tiptap/vue-3'
import {
  ChevronDown,
  Bold,
  Italic,
  Bot,
  List,
  ListOrdered,
  Link as LinkIcon,
  Check,
  X
} from 'lucide-vue-next'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import Placeholder from '@tiptap/extension-placeholder'
import Image from '@tiptap/extension-image'
import StarterKit from '@tiptap/starter-kit'
import Link from '@tiptap/extension-link'
import Table from '@tiptap/extension-table'
import TableRow from '@tiptap/extension-table-row'
import TableCell from '@tiptap/extension-table-cell'
import TableHeader from '@tiptap/extension-table-header'

const selectedText = defineModel('selectedText', { default: '' })
const textContent = defineModel('textContent')
const htmlContent = defineModel('htmlContent')
const isBold = defineModel('isBold')
const isItalic = defineModel('isItalic')
const cursorPosition = defineModel('cursorPosition', { default: 0 })
const showLinkInput = ref(false)
const linkUrl = ref('')

const props = defineProps({
  placeholder: String,
  contentToSet: String,
  setInlineImage: Object,
  insertContent: String,
  clearContent: Boolean,
  autoFocus: {
    type: Boolean,
    default: true
  },
  aiPrompts: {
    type: Array,
    default: () => []
  }
})

const emit = defineEmits(['send', 'aiPromptSelected'])

const emitPrompt = (key) => emit('aiPromptSelected', key)

const getSelectionText = (from, to, doc) => doc.textBetween(from, to)

// To preseve the table styling in emails, need to set the table style inline.
// Created these custom extensions to set the table style inline.
const CustomTable = Table.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') + ' border: 1px solid #dee2e6 !important; width: 100%; margin:0; table-layout: fixed; border-collapse: collapse; position:relative; border-radius: 0.25rem;'
      }
    }
  }
})

const CustomTableCell = TableCell.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          ' border: 1px solid #dee2e6 !important; box-sizing: border-box !important; min-width: 1em !important; padding: 6px 8px !important; vertical-align: top !important;'
      }
    }
  }
})

const CustomTableHeader = TableHeader.extend({
  addAttributes() {
    return {
      ...this.parent?.(),
      style: {
        parseHTML: (element) =>
          (element.getAttribute('style') || '') +
          ' background-color: #f8f9fa !important; color: #212529 !important; font-weight: bold !important; text-align: left !important; border: 1px solid #dee2e6 !important; padding: 6px 8px !important;'
      }
    }
  }
})

const editorConfig = computed(() => ({
  extensions: [
    StarterKit.configure(),
    Image.configure({ HTMLAttributes: { class: 'inline-image' } }),
    Placeholder.configure({ placeholder: () => props.placeholder }),
    Link,
    CustomTable.configure({
      resizable: false
    }),
    TableRow,
    CustomTableCell,
    CustomTableHeader
  ],
  autofocus: props.autoFocus,
  editorProps: {
    attributes: { class: 'outline-none' },
    handleKeyDown: (view, event) => {
      if (event.ctrlKey && event.key === 'Enter') {
        emit('send')
        return true
      }
      if (event.ctrlKey && event.key.toLowerCase() === 'b') {
        // Prevent outer listeners
        event.stopPropagation()
        return false
      }
    }
  }
}))

const editor = ref(
  useEditor({
    ...editorConfig.value,
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

const toggleBulletList = () => {
  if (editor.value) {
    editor.value.chain().focus().toggleBulletList().run()
  }
}

const toggleOrderedList = () => {
  if (editor.value) {
    editor.value.chain().focus().toggleOrderedList().run()
  }
}

const openLinkModal = () => {
  if (editor.value?.isActive('link')) {
    linkUrl.value = editor.value.getAttributes('link').href
  } else {
    linkUrl.value = ''
  }
  showLinkInput.value = true
}

const setLink = () => {
  if (linkUrl.value) {
    editor.value?.chain().focus().extendMarkRange('link').setLink({ href: linkUrl.value }).run()
  }
  showLinkInput.value = false
}

const unsetLink = () => {
  editor.value?.chain().focus().unsetLink().run()
  showLinkInput.value = false
}
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

.tiptap {
  // Table styling
  .tableWrapper {
    margin: 1.5rem 0;
    overflow-x: auto;
  }

  // Anchor tag styling
  a {
    color: #0066cc;
    cursor: pointer;

    &:hover {
      color: #003d7a;
    }
  }
}
</style>

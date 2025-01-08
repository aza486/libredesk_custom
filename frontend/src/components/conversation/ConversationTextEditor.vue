<template>
  <div class="max-h-[600px] overflow-y-auto">
    <BubbleMenu :editor="editor" :tippy-options="{ duration: 100 }" v-if="editor">
      <div class="BubbleMenu">
        <DropdownMenu>
          <DropdownMenuTrigger>
            <Button size="sm" variant="outline">
              AI
              <ChevronDown class="w-4 h-4 ml-2" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem>Make friendly</DropdownMenuItem>
            <DropdownMenuItem>Make formal</DropdownMenuItem>
            <DropdownMenuItem>Make casual</DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </BubbleMenu>
    <EditorContent :editor="editor" />
  </div>
</template>

<script setup>
import { ref, watch, watchEffect, onUnmounted } from 'vue'
import { useEditor, EditorContent, BubbleMenu } from '@tiptap/vue-3'
import { ChevronDown } from 'lucide-vue-next';
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

const emit = defineEmits([
  'send',
  'editorText',
  'updateBold',
  'updateItalic',
  'contentCleared',
  'contentSet',
  'editorReady'
])

const props = defineProps({
  placeholder: String,
  isBold: Boolean,
  isItalic: Boolean,
  clearContent: Boolean,
  contentToSet: String
})

const editor = ref(
  useEditor({
    content: '',
    extensions: [
      StarterKit,
      Image.configure({
        HTMLAttributes: {
          // Common class for all inline images.
          class: 'inline-image',
        },
      }),
      Placeholder.configure({
        placeholder: () => {
          return props.placeholder
        }
      }),
      Link,
    ],
    autofocus: true,
    editorProps: {
      attributes: {
        // No outline for the editor.
        class: 'outline-none'
      },
    }
  })
)

watchEffect(() => {
  if (editor.value) {
    // Emit the editor instance when it's ready
    if (editor.value) {
      emit('editorReady', editor.value)
    }

    emit('editorText', {
      text: editor.value.getText(),
      html: editor.value.getHTML()
    })

    // Emit bold and italic state changes
    emit('updateBold', editor.value.isActive('bold'))
    emit('updateItalic', editor.value.isActive('italic'))
  }
})

// Watcher for bold and italic changes
watchEffect(() => {
  if (props.isBold !== editor.value?.isActive('bold')) {
    if (props.isBold) {
      editor.value?.chain().focus().setBold().run()
    } else {
      editor.value?.chain().focus().unsetBold().run()
    }
  }
  if (props.isItalic !== editor.value?.isActive('italic')) {
    if (props.isItalic) {
      editor.value?.chain().focus().setItalic().run()
    } else {
      editor.value?.chain().focus().unsetItalic().run()
    }
  }
})

// Watcher for clearContent prop
watchEffect(() => {
  if (props.clearContent) {
    editor.value?.commands.clearContent()
    emit('contentCleared')
  }
})

watch(
  () => props.contentToSet,
  (newContent) => {
    if (newContent) {
      // Remove trailing break when setting content
      editor.value.commands.setContent(newContent)
      editor.value.commands.focus()
      emit('contentSet')
    }
  }
)

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

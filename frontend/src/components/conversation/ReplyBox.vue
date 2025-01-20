<template>
  <div>
    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = false">
      <DialogContent
        class="max-w-[80%] max-h-[80%] h-[80%] w-full m-0 p-6"
        @escapeKeyDown="isEditorFullscreen = false"
      >
        <div v-if="isEditorFullscreen" class="h-full flex flex-col fullscreen-tiptap-editor">
          <!-- Message type toggle -->
          <div class="flex justify-between px-2 border-b py-2">
            <Tabs v-model="messageType">
              <TabsList>
                <TabsTrigger value="reply"> Reply </TabsTrigger>
                <TabsTrigger value="private_note"> Private note </TabsTrigger>
              </TabsList>
            </Tabs>
            <div
              class="flex items-center mr-2 cursor-pointer"
              @click="isEditorFullscreen = !isEditorFullscreen"
            >
              <!-- <Minimize2 size="16" /> -->
            </div>
          </div>

          <!-- Main Editor -->
          <div class="flex-grow overflow-y-auto">
            <Editor
              v-model:selectedText="selectedText"
              v-model:isBold="isBold"
              v-model:isItalic="isItalic"
              v-model:htmlContent="htmlContent"
              v-model:textContent="textContent"
              :placeholder="editorPlaceholder"
              :aiPrompts="aiPrompts"
              @aiPromptSelected="handleAiPromptSelected"
              :contentToSet="contentToSet"
              @send="handleSend"
              v-model:cursorPosition="cursorPosition"
              :clearContent="clearEditorContent"
              :setInlineImage="setInlineImage"
              :insertContent="insertContent"
              class="h-full"
            />
          </div>

          <!-- Macro preview -->
          <MacroActionsPreview
            v-if="conversationStore.conversation?.macro?.actions?.length > 0"
            :actions="conversationStore.conversation.macro.actions"
            :onRemove="conversationStore.removeMacroAction"
          />

          <!-- Attachments preview -->
          <AttachmentsPreview
            :attachments="attachments"
            :onDelete="handleOnFileDelete"
            v-if="attachments.length > 0"
          />

          <!-- Bottom menu bar -->
          <ReplyBoxBottomMenuBar
            class="mt-1"
            :handleFileUpload="handleFileUpload"
            :handleInlineImageUpload="handleInlineImageUpload"
            :isBold="isBold"
            :isItalic="isItalic"
            @toggleBold="toggleBold"
            @toggleItalic="toggleItalic"
            :enableSend="enableSend"
            :handleSend="handleSend"
            @emojiSelect="handleEmojiSelect"
          >
          </ReplyBoxBottomMenuBar>
        </div>
      </DialogContent>
    </Dialog>

    <div class="m-3 border rounded-xl box">
      <!-- Main Editor non-fullscreen -->
      <div v-if="!isEditorFullscreen">
        <!-- Message type toggle -->
        <div class="flex justify-between px-2 border-b py-2">
          <Tabs v-model="messageType">
            <TabsList>
              <TabsTrigger value="reply"> Reply </TabsTrigger>
              <TabsTrigger value="private_note"> Private note </TabsTrigger>
            </TabsList>
          </Tabs>
          <div
            class="flex items-center mr-2 cursor-pointer"
            @click="isEditorFullscreen = !isEditorFullscreen"
          >
            <Maximize2 size="16" />
          </div>
        </div>

        <!-- Main Editor -->
        <Editor
          v-model:selectedText="selectedText"
          v-model:isBold="isBold"
          v-model:isItalic="isItalic"
          v-model:htmlContent="htmlContent"
          v-model:textContent="textContent"
          :placeholder="editorPlaceholder"
          :aiPrompts="aiPrompts"
          @aiPromptSelected="handleAiPromptSelected"
          :contentToSet="contentToSet"
          @send="handleSend"
          v-model:cursorPosition="cursorPosition"
          :clearContent="clearEditorContent"
          :setInlineImage="setInlineImage"
          :insertContent="insertContent"
        />

        <!-- Macro preview -->
        <MacroActionsPreview
          v-if="conversationStore.conversation?.macro?.actions?.length > 0"
          :actions="conversationStore.conversation.macro.actions"
          :onRemove="conversationStore.removeMacroAction"
        />

        <!-- Attachments preview -->
        <AttachmentsPreview
          :attachments="attachments"
          :onDelete="handleOnFileDelete"
          v-if="attachments.length > 0"
        />

        <!-- Bottom menu bar -->
        <ReplyBoxBottomMenuBar
          class="mt-1"
          :handleFileUpload="handleFileUpload"
          :handleInlineImageUpload="handleInlineImageUpload"
          :isBold="isBold"
          :isItalic="isItalic"
          @toggleBold="toggleBold"
          @toggleItalic="toggleItalic"
          :enableSend="enableSend"
          :handleSend="handleSend"
          @emojiSelect="handleEmojiSelect"
        >
        </ReplyBoxBottomMenuBar>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, nextTick, watch } from 'vue'
import { transformImageSrcToCID } from '@/utils/strings'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { Maximize2 } from 'lucide-vue-next'
import api from '@/api'

import Editor from './ConversationTextEditor.vue'
import { useConversationStore } from '@/stores/conversation'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useEmitter } from '@/composables/useEmitter'
import AttachmentsPreview from '@/components/attachment/AttachmentsPreview.vue'
import MacroActionsPreview from '../macro/MacroActionsPreview.vue'
import ReplyBoxBottomMenuBar from '@/components/conversation/ReplyBoxMenuBar.vue'

const conversationStore = useConversationStore()
const emitter = useEmitter()
const insertContent = ref(null)
const setInlineImage = ref(null)
const clearEditorContent = ref(false)
const isEditorFullscreen = ref(false)
const cursorPosition = ref(0)
const selectedText = ref('')
const htmlContent = ref('')
const textContent = ref('')
const contentToSet = ref('')
const isBold = ref(false)
const isItalic = ref(false)
const messageType = ref('reply')

const aiPrompts = ref([])

const editorPlaceholder = 'Press Enter to add a new line; Press Ctrl + Enter to send.'

onMounted(async () => {
  await fetchAiPrompts()
})

const fetchAiPrompts = async () => {
  try {
    const resp = await api.getAiPrompts()
    aiPrompts.value = resp.data.data
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const handleAiPromptSelected = async (key) => {
  try {
    const resp = await api.aiCompletion({
      prompt_key: key,
      content: selectedText.value
    })
    contentToSet.value = resp.data.data.replace(/\n/g, '<br>')
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const toggleBold = () => {
  isBold.value = !isBold.value
}

const toggleItalic = () => {
  isItalic.value = !isItalic.value
}

const attachments = computed(() => {
  return conversationStore.conversation.mediaFiles.filter(
    (upload) => upload.disposition === 'attachment'
  )
})

const enableSend = computed(() => {
  return textContent.value.trim().length > 0 ||
    conversationStore.conversation?.macro?.actions?.length > 0
    ? true
    : false
})

const hasTextContent = computed(() => {
  return textContent.value.trim().length > 0
})

const handleFileUpload = (event) => {
  for (const file of event.target.files) {
    api
      .uploadMedia({
        files: file,
        inline: false,
        linked_model: 'messages'
      })
      .then((resp) => {
        conversationStore.conversation.mediaFiles.push(resp.data.data)
      })
      .catch((error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          title: 'Error',
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      })
  }
}

const handleInlineImageUpload = (event) => {
  for (const file of event.target.files) {
    api
      .uploadMedia({
        files: file,
        inline: true,
        linked_model: 'messages'
      })
      .then((resp) => {
        setInlineImage.value = {
          src: resp.data.data.url,
          alt: resp.data.data.filename,
          title: resp.data.data.uuid
        }
        conversationStore.conversation.mediaFiles.push(resp.data.data)
      })
      .catch((error) => {
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          title: 'Error',
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      })
  }
}

const handleSend = async () => {
  isEditorFullscreen.value = false
  try {
    // Send message if there is text content in the editor.
    if (hasTextContent.value) {
      // Replace inline image url with cid.
      const message = transformImageSrcToCID(htmlContent.value)

      // Check which images are still in editor before sending.
      const parser = new DOMParser()
      const doc = parser.parseFromString(htmlContent.value, 'text/html')
      const inlineImageUUIDs = Array.from(doc.querySelectorAll('img.inline-image'))
        .map((img) => img.getAttribute('title'))
        .filter(Boolean)

      conversationStore.conversation.mediaFiles = conversationStore.conversation.mediaFiles.filter(
        (file) =>
          // Keep if:
          // 1. Not an inline image OR
          // 2. Is an inline image that exists in editor
          file.disposition !== 'inline' || inlineImageUUIDs.includes(file.uuid)
      )

      await api.sendMessage(conversationStore.current.uuid, {
        private: messageType.value === 'private_note',
        message: message,
        attachments: conversationStore.conversation.mediaFiles.map((file) => file.id)
      })
    }

    // Apply macro if it exists.
    if (conversationStore.conversation?.macro?.actions?.length > 0) {
      await api.applyMacro(
        conversationStore.current.uuid,
        conversationStore.conversation.macro.id,
        conversationStore.conversation.macro.actions
      )
    }
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    clearEditorContent.value = true
    conversationStore.resetMacro()
    conversationStore.resetMediaFiles()
    nextTick(() => {
      clearEditorContent.value = false
    })
  }
  api.updateAssigneeLastSeen(conversationStore.current.uuid)
}

const handleOnFileDelete = (uuid) => {
  conversationStore.conversation.mediaFiles = conversationStore.conversation.mediaFiles.filter(
    (item) => item.uuid !== uuid
  )
}

const handleEmojiSelect = (emoji) => {
  insertContent.value = undefined
  // Force reactivity so the user can select the same emoji multiple times
  nextTick(() => (insertContent.value = emoji))
}

// Watch for changes in macro content and update editor content.
watch(
  () => conversationStore.conversation.macro,
  () => {
    // hack: Quill editor adds <p><br></p> replace with <p></p>
    if (conversationStore.conversation?.macro?.message_content) {
      contentToSet.value = conversationStore.conversation.macro.message_content.replace(
        /<p><br><\/p>/g,
        '<p></p>'
      )
    }
  },
  { deep: true }
)
</script>

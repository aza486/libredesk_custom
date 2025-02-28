<template>
  <div class="text-foreground bg-background">
    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = false">
      <DialogContent
        class="max-w-[70%] max-h-[70%] h-[70%] bg-card text-card-foreground p-4 flex flex-col"
        @escapeKeyDown="isEditorFullscreen = false"
        :hide-close-button="true"
      >
        <ReplyBoxContent
          v-if="isEditorFullscreen"
          :isFullscreen="true"
          :aiPrompts="aiPrompts"
          :isSending="isSending"
          :uploadingFiles="uploadingFiles"
          :clearEditorContent="clearEditorContent"
          :htmlContent="htmlContent"
          :textContent="textContent"
          :selectedText="selectedText"
          :isBold="isBold"
          :isItalic="isItalic"
          :cursorPosition="cursorPosition"
          :contentToSet="contentToSet"
          :cc="cc"
          :bcc="bcc"
          :emailErrors="emailErrors"
          :messageType="messageType"
          :showBcc="showBcc"
          @update:htmlContent="htmlContent = $event"
          @update:textContent="textContent = $event"
          @update:selectedText="selectedText = $event"
          @update:isBold="isBold = $event"
          @update:isItalic="isItalic = $event"
          @update:cursorPosition="cursorPosition = $event"
          @toggleFullscreen="isEditorFullscreen = false"
          @update:messageType="messageType = $event"
          @update:cc="cc = $event"
          @update:bcc="bcc = $event"
          @update:showBcc="showBcc = $event"
          @updateEmailErrors="emailErrors = $event"
          @send="processSend"
          @fileUpload="handleFileUpload"
          @inlineImageUpload="handleInlineImageUpload"
          @fileDelete="handleOnFileDelete"
          @aiPromptSelected="handleAiPromptSelected"
          class="h-full flex-grow"
        />
      </DialogContent>
    </Dialog>

    <!-- Main Editor non-fullscreen -->
    <div
      class="bg-card text-card-foreground box m-2 px-2 pt-2 flex flex-col"
      v-if="!isEditorFullscreen"
    >
      <ReplyBoxContent
        :isFullscreen="false"
        :aiPrompts="aiPrompts"
        :isSending="isSending"
        :uploadingFiles="uploadingFiles"
        :clearEditorContent="clearEditorContent"
        :htmlContent="htmlContent"
        :textContent="textContent"
        :selectedText="selectedText"
        :isBold="isBold"
        :isItalic="isItalic"
        :cursorPosition="cursorPosition"
        :contentToSet="contentToSet"
        :cc="cc"
        :bcc="bcc"
        :emailErrors="emailErrors"
        :messageType="messageType"
        :showBcc="showBcc"
        @update:htmlContent="htmlContent = $event"
        @update:textContent="textContent = $event"
        @update:selectedText="selectedText = $event"
        @update:isBold="isBold = $event"
        @update:isItalic="isItalic = $event"
        @update:cursorPosition="cursorPosition = $event"
        @toggleFullscreen="isEditorFullscreen = true"
        @update:messageType="messageType = $event"
        @update:cc="cc = $event"
        @update:bcc="bcc = $event"
        @update:showBcc="showBcc = $event"
        @updateEmailErrors="emailErrors = $event"
        @send="processSend"
        @fileUpload="handleFileUpload"
        @inlineImageUpload="handleInlineImageUpload"
        @fileDelete="handleOnFileDelete"
        @aiPromptSelected="handleAiPromptSelected"
      />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, nextTick, watch, computed } from 'vue'
import { transformImageSrcToCID } from '@/utils/strings'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

import { useConversationStore } from '@/stores/conversation'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { useEmitter } from '@/composables/useEmitter'
import ReplyBoxContent from '@/features/conversation/ReplyBoxContent.vue'

const conversationStore = useConversationStore()
const emitter = useEmitter()

// Shared state between the two editor components.
const clearEditorContent = ref(false)
const isEditorFullscreen = ref(false)
const isSending = ref(false)
const messageType = ref('reply')
const cc = ref('')
const bcc = ref('')
const showBcc = ref(false)
const emailErrors = ref([])
const aiPrompts = ref([])
const uploadingFiles = ref([])
const htmlContent = ref('')
const textContent = ref('')
const selectedText = ref('')
const isBold = ref(false)
const isItalic = ref(false)
const cursorPosition = ref(0)
const contentToSet = ref('')

onMounted(async () => {
  await fetchAiPrompts()
})

/**
 * Fetches AI prompts from the server.
 */
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

/**
 * Handles the AI prompt selection event.
 * Sends the selected prompt key and the current text content to the server for completion.
 * Sets the response as the new content in the editor.
 * @param {String} key - The key of the selected AI prompt
 */
const handleAiPromptSelected = async (key) => {
  try {
    const resp = await api.aiCompletion({
      prompt_key: key,
      content: textContent.value
    })
    contentToSet.value = JSON.stringify({
      content: resp.data.data.replace(/\n/g, '<br>'),
      timestamp: Date.now()
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

/**
 * Handles the file upload process when files are selected.
 * Uploads each file to the server and adds them to the conversation's mediaFiles.
 * @param {Event} event - The file input change event containing selected files
 */
const handleFileUpload = (event) => {
  const files = Array.from(event.target.files)
  uploadingFiles.value = files

  for (const file of files) {
    api
      .uploadMedia({
        files: file,
        inline: false,
        linked_model: 'messages'
      })
      .then((resp) => {
        conversationStore.conversation.mediaFiles.push(resp.data.data)
        uploadingFiles.value = uploadingFiles.value.filter((f) => f.name !== file.name)
      })
      .catch((error) => {
        uploadingFiles.value = uploadingFiles.value.filter((f) => f.name !== file.name)
        emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
          title: 'Error',
          variant: 'destructive',
          description: handleHTTPError(error).message
        })
      })
  }
}

// Inline image upload is not supported yet.
const handleInlineImageUpload = (event) => {
  for (const file of event.target.files) {
    api
      .uploadMedia({
        files: file,
        inline: true,
        linked_model: 'messages'
      })
      .then((resp) => {
        const imageData = {
          src: resp.data.data.url,
          alt: resp.data.data.filename,
          title: resp.data.data.uuid
        }
        conversationStore.conversation.mediaFiles.push(resp.data.data)
        return imageData
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

/**
 * Returns true if the editor has text content.
 */
const hasTextContent = computed(() => {
  return textContent.value.trim().length > 0
})

/**
 * Processes the send action.
 */
const processSend = async () => {
  isEditorFullscreen.value = false
  try {
    isSending.value = true

    // Send message if there is text content in the editor.
    if (hasTextContent.value > 0) {
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
        attachments: conversationStore.conversation.mediaFiles.map((file) => file.id),
        // Convert email addresses to array and remove empty strings.
        cc: cc.value
          .split(',')
          .map((email) => email.trim())
          .filter((email) => email),
        bcc: bcc.value
          ? bcc.value
              .split(',')
              .map((email) => email.trim())
              .filter((email) => email)
          : []
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
    isSending.value = false
    clearEditorContent.value = true
    // Reset media and macro in conversation store.
    conversationStore.resetMacro()
    conversationStore.resetMediaFiles()
    emailErrors.value = []
    nextTick(() => {
      clearEditorContent.value = false
    })
  }
  // Update assignee last seen timestamp.
  api.updateAssigneeLastSeen(conversationStore.current.uuid)
}

/**
 * Handles the file delete event.
 * Removes the file from the conversation's mediaFiles.
 * @param {String} uuid - The UUID of the file to delete
 */
const handleOnFileDelete = (uuid) => {
  conversationStore.conversation.mediaFiles = conversationStore.conversation.mediaFiles.filter(
    (item) => item.uuid !== uuid
  )
}

/**
 * Watches for changes in the conversation's macro and updates the editor content with the macro content.
 */
watch(
  () => conversationStore.conversation.macro,
  () => {
    // hack: Quill editor adds <p><br></p> replace with <p></p>
    // Maybe use some other editor that doesn't add this?
    if (conversationStore.conversation?.macro?.message_content) {
      const contentToRender = conversationStore.conversation.macro.message_content.replace(
        /<p><br><\/p>/g,
        '<p></p>'
      )
      // Add timestamp to ensure the watcher detects the change even for identical content,
      // As user can send the same macro multiple times.
      contentToSet.value = JSON.stringify({
        content: contentToRender,
        timestamp: Date.now()
      })
    }
  },
  { deep: true }
)

// Initialize cc and bcc from conversation store
watch(
  () => conversationStore.currentCC,
  (newVal) => {
    cc.value = newVal?.join(', ') || ''
  },
  { deep: true, immediate: true }
)

watch(
  () => conversationStore.currentBCC,
  (newVal) => {
    const newBcc = newVal?.join(', ') || ''
    bcc.value = newBcc
    // Only show BCC field if it has content
    if (newBcc.length > 0) {
      showBcc.value = true
    }
  },
  { deep: true, immediate: true }
)
</script>

<template>
  <div class="text-foreground bg-background">
    <!-- Fullscreen editor -->
    <Dialog :open="isEditorFullscreen" @update:open="isEditorFullscreen = false">
      <DialogContent
        class="max-w-[70%] max-h-[70%] h-[90%] w-full bg-card text-card-foreground rounded-lg shadow-xl p-0 px-4 py-4"
        @escapeKeyDown="isEditorFullscreen = false"
        hide-close-button="true"
      >
        <div v-if="isEditorFullscreen" class="h-full flex flex-col">
          <!-- Message type toggle -->
          <div class="flex justify-between items-center border-b border-border pb-4">
            <Tabs v-model="messageType" class="space-x-2">
              <TabsList class="bg-muted p-1 rounded-md">
                <TabsTrigger
                  value="reply"
                  class="px-3 py-1 rounded-md transition-colors duration-200"
                  :class="{ 'bg-background text-foreground': messageType === 'reply' }"
                >
                  Reply
                </TabsTrigger>
                <TabsTrigger
                  value="private_note"
                  class="px-3 py-1 rounded-md transition-colors duration-200"
                  :class="{ 'bg-background text-foreground': messageType === 'private_note' }"
                >
                  Private note
                </TabsTrigger>
              </TabsList>
            </Tabs>
            <span
              class="text-muted-foreground hover:text-foreground transition-colors duration-200 cursor-pointer"
              variant="ghost"
              @click="isEditorFullscreen = false"
            >
              <Minimize2 size="18" />
            </span>
          </div>

          <!-- CC and BCC fields -->
          <div class="space-y-3 p-4 border-b border-border" v-if="messageType === 'reply'">
            <div class="flex items-center space-x-2">
              <label class="w-12 text-sm font-medium text-muted-foreground">CC:</label>
              <Input
                type="text"
                placeholder="Email addresses separated by comma"
                v-model="cc"
                class="flex-grow px-3 py-2 text-sm border rounded-md focus:ring-2 focus:ring-ring"
                @blur="validateEmails('cc')"
              />
              <Button
                size="sm"
                @click="hideBcc"
                class="text-sm bg-secondary text-secondary-foreground hover:bg-secondary/80"
              >
                {{ showBcc ? 'Remove BCC' : 'BCC' }}
              </Button>
            </div>
            <div v-if="showBcc" class="flex items-center space-x-2">
              <label class="w-12 text-sm font-medium text-muted-foreground">BCC:</label>
              <Input
                type="text"
                placeholder="Email addresses separated by comma"
                v-model="bcc"
                class="flex-grow px-3 py-2 text-sm border rounded-md focus:ring-2 focus:ring-ring"
                @blur="validateEmails('bcc')"
              />
            </div>
          </div>

          <div
            v-if="emailErrors.length > 0"
            class="mb-4 px-2 py-1 bg-destructive/10 border border-destructive text-destructive rounded"
          >
            <p v-for="error in emailErrors" :key="error" class="text-sm">{{ error }}</p>
          </div>

          <!-- Main Editor -->
          <div class="flex-grow overflow-y-auto p-2">
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
            class="mt-4"
          />

          <!-- Attachments preview -->
          <AttachmentsPreview
            :attachments="attachments"
            :uploadingFiles="uploadingFiles"
            :onDelete="handleOnFileDelete"
            v-if="attachments.length > 0 || uploadingFiles.length > 0"
            class="mt-4"
          />

          <!-- Bottom menu bar -->
          <ReplyBoxBottomMenuBar
            class="mt-4 border-t border-border pt-4"
            :handleFileUpload="handleFileUpload"
            :handleInlineImageUpload="handleInlineImageUpload"
            :isBold="isBold"
            :isItalic="isItalic"
            @toggleBold="toggleBold"
            @toggleItalic="toggleItalic"
            :enableSend="enableSend"
            :handleSend="handleSend"
            @emojiSelect="handleEmojiSelect"
          />
        </div>
      </DialogContent>
    </Dialog>

    <div class="bg-card text-card-foreground rounded-lg shadow-md px-2 border pt-2 m-2">
      <!-- Main Editor non-fullscreen -->
      <div v-if="!isEditorFullscreen" class="">
        <!-- Message type toggle -->
        <div class="flex justify-between items-center mb-4">
          <Tabs v-model="messageType">
            <TabsList class="bg-muted p-1 rounded-md">
              <TabsTrigger
                value="reply"
                class="px-3 py-1 rounded-md transition-colors duration-200"
                :class="{ 'bg-background text-foreground': messageType === 'reply' }"
              >
                Reply
              </TabsTrigger>
              <TabsTrigger
                value="private_note"
                class="px-3 py-1 rounded-md transition-colors duration-200"
                :class="{ 'bg-background text-foreground': messageType === 'private_note' }"
              >
                Private note
              </TabsTrigger>
            </TabsList>
          </Tabs>
          <span
            class="text-muted-foreground hover:text-foreground transition-colors duration-200 cursor-pointer mr-2"
            variant="ghost"
            @click="isEditorFullscreen = true"
          >
            <Maximize2 size="18" />
          </span>
        </div>

        <div class="space-y-3 mb-4" v-if="messageType === 'reply'">
          <div class="flex items-center space-x-2">
            <label class="w-12 text-sm font-medium text-muted-foreground">CC:</label>
            <Input
              type="text"
              placeholder="Email addresses separated by comma"
              v-model="cc"
              class="flex-grow px-3 py-2 text-sm border rounded-md focus:ring-2 focus:ring-ring"
              @blur="validateEmails('cc')"
            />
            <Button
              size="sm"
              @click="hideBcc"
              class="text-sm bg-secondary text-secondary-foreground hover:bg-secondary/80"
            >
              {{ showBcc ? 'Remove BCC' : 'BCC' }}
            </Button>
          </div>
          <div v-if="showBcc" class="flex items-center space-x-2">
            <label class="w-12 text-sm font-medium text-muted-foreground">BCC:</label>
            <Input
              type="text"
              placeholder="Email addresses separated by comma"
              v-model="bcc"
              class="flex-grow px-3 py-2 text-sm border rounded-md focus:ring-2 focus:ring-ring"
              @blur="validateEmails('bcc')"
            />
          </div>
        </div>

        <div
          v-if="emailErrors.length > 0"
          class="mb-4 px-2 py-1 bg-destructive/10 border border-destructive text-destructive rounded"
        >
          <p v-for="error in emailErrors" :key="error" class="text-sm">{{ error }}</p>
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
          :uploadingFiles="uploadingFiles"
          :onDelete="handleOnFileDelete"
          v-if="attachments.length > 0 || uploadingFiles.length > 0"
          class="mt-4"
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
        />
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed, nextTick, watch } from 'vue'
import { transformImageSrcToCID } from '@/utils/strings'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { Maximize2, Minimize2 } from 'lucide-vue-next'
import api from '@/api'

import Editor from './ConversationTextEditor.vue'
import { useConversationStore } from '@/stores/conversation'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useEmitter } from '@/composables/useEmitter'
import AttachmentsPreview from '@/features/conversation/message/attachment/AttachmentsPreview.vue'
import MacroActionsPreview from '@/features/conversation/MacroActionsPreview.vue'
import ReplyBoxBottomMenuBar from '@/features/conversation/ReplyBoxMenuBar.vue'

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
const showBcc = ref(false)
const cc = ref('')
const bcc = ref('')
const emailErrors = ref([])
const aiPrompts = ref([])
const uploadingFiles = ref([])
const editorPlaceholder = 'Press Enter to add a new line; Press Ctrl + Enter to send.'

onMounted(async () => {
  await fetchAiPrompts()
})

const hideBcc = () => {
  showBcc.value = !showBcc.value
}

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
    if (newBcc.length == 0) {
      showBcc.value = false
    } else {
      showBcc.value = true
    }
  },
  { deep: true, immediate: true }
)

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
  return (
    (textContent.value.trim().length > 0 ||
      conversationStore.conversation?.macro?.actions?.length > 0) &&
    emailErrors.value.length === 0
  )
})

const hasTextContent = computed(() => {
  return textContent.value.trim().length > 0
})

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

const validateEmails = (field) => {
  const emails = field === 'cc' ? cc.value : bcc.value
  const emailList = emails
    .split(',')
    .map((e) => e.trim())
    .filter((e) => e !== '')
  const emailRegex = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
  const invalidEmails = emailList.filter((email) => !emailRegex.test(email))

  // Remove any existing errors for this field
  emailErrors.value = emailErrors.value.filter(
    (error) => !error.startsWith(`Invalid email(s) in ${field.toUpperCase()}`)
  )

  // Add new error if there are invalid emails
  if (invalidEmails.length > 0) {
    emailErrors.value.push(
      `Invalid email(s) in ${field.toUpperCase()}: ${invalidEmails.join(', ')}`
    )
  }
}

const handleSend = async () => {
  if (emailErrors.value.length > 0) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: 'Please correct the email errors before sending.'
    })
    return
  }

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
        attachments: conversationStore.conversation.mediaFiles.map((file) => file.id),
        // Convert email addresses to array and remove empty strings.
        cc: cc.value
          .split(',')
          .map((email) => email.trim())
          .filter((email) => email),
        bcc: showBcc.value
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
    clearEditorContent.value = true
    conversationStore.resetMacro()
    conversationStore.resetMediaFiles()
    emailErrors.value = []
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

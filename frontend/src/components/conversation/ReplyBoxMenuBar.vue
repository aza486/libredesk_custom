<template>
  <div class="flex justify-between items-center border-t h-14 px-2 relative">
    <EmojiPicker v-if="isEmojiPickerVisible" ref="emojiPickerRef" :native="true" @select="onSelectEmoji"
      class="absolute bottom-14 left-14" />
    <div class="flex justify-items-start gap-2">
      <!-- File inputs -->
      <input type="file" class="hidden" ref="attachmentInput" multiple @change="handleFileUpload" />
      <input type="file" class="hidden" ref="inlineImageInput" accept="image/*" @change="handleInlineImageUpload" />
      <!-- Editor buttons -->
      <Toggle class="px-2 py-2 border-0" variant="outline" @click="triggerFileUpload" :pressed="false">
        <Paperclip class="h-4 w-4" />
      </Toggle>
      <Toggle class="px-2 py-2 border-0" variant="outline" @click="toggleEmojiPicker" :pressed="isEmojiPickerVisible">
        <Smile class="h-4 w-4" />
      </Toggle>
    </div>
    <Button class="h-8 w-6 px-8" @click="handleSend" :disabled="!hasText">Send</Button>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { onClickOutside } from '@vueuse/core'
import { Button } from '@/components/ui/button'
import { Toggle } from '@/components/ui/toggle'
import { Paperclip, Smile } from 'lucide-vue-next'
import EmojiPicker from 'vue3-emoji-picker'
import 'vue3-emoji-picker/css'

const attachmentInput = ref(null)
const inlineImageInput = ref(null)
const isEmojiPickerVisible = ref(false)
const emojiPickerRef = ref(null)
const emit = defineEmits(['toggleBold', 'toggleItalic', 'emojiSelect'])

defineProps({
  isBold: Boolean,
  isItalic: Boolean,
  hasText: Boolean,
  handleSend: Function,
  handleFileUpload: Function,
  handleInlineImageUpload: Function
})

onClickOutside(emojiPickerRef, () => {
  isEmojiPickerVisible.value = false
})

const triggerFileUpload = () => {
  attachmentInput.value.click()
}

const toggleEmojiPicker = () => {
  isEmojiPickerVisible.value = !isEmojiPickerVisible.value
}

function onSelectEmoji (emoji) {
  emit('emojiSelect', emoji.i)
}
</script>

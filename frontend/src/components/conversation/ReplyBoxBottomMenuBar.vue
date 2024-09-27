<template>
  <div class="flex justify-between items-center border-y h-14 px-2">
    <div class="flex justify-items-start gap-2">
      <input type="file" class="hidden" ref="attachmentInput" multiple @change="handleFileUpload" />
      <input type="file" class="hidden" ref="inlineImageInput" accept="image/*" @change="handleInlineImageUpload" />
      <Toggle class="px-2 py-2 border-0" variant="outline" @click="toggleBold" :pressed="isBold">
        <Bold class="h-4 w-4" />
      </Toggle>
      <Toggle class="px-2 py-2 border-0" variant="outline" @click="toggleItalic" :pressed="isItalic">
        <Italic class="h-4 w-4" />
      </Toggle>
      <Toggle class="px-2 py-2 border-0" variant="outline" @click="triggerFileUpload" :pressed="false">
        <Paperclip class="h-4 w-4" />
      </Toggle>
      <Toggle class="px-2 py-2 border-0" variant="outline" @click="triggerInlineImage" :pressed="false">
        <Image class="h-4 w-4" />
      </Toggle>
    </div>
    <Button class="h-8 w-6 px-8" @click="handleSend" :disabled="!hasText"> Send
    </Button>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Button } from '@/components/ui/button'
import { Toggle } from '@/components/ui/toggle'
import { Paperclip, Bold, Italic, Image } from 'lucide-vue-next'

const attachmentInput = ref(null)
const inlineImageInput = ref(null)
const emit = defineEmits(['toggleBold', 'toggleItalic'])
defineProps({
  isBold: Boolean,
  isItalic: Boolean,
  hasText: Boolean,
  handleSend: Function,
  handleFileUpload: Function,
  handleInlineImageUpload: Function
})

const toggleBold = () => {
  emit('toggleBold')
}

const toggleItalic = () => {
  emit('toggleItalic')
}

const triggerFileUpload = () => {
  attachmentInput.value.click()
}

const triggerInlineImage = () => {
  inlineImageInput.value.click()
}
</script>

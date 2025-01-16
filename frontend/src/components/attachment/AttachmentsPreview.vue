<template>
  <div class="flex flex-wrap gap-2 px-2 py-1">
    <TransitionGroup name="attachment-list" tag="div" class="flex flex-wrap gap-2">
      <div
        v-for="attachment in attachments"
        :key="attachment.uuid"
        class="flex items-center bg-white border border-gray-200 rounded shadow-sm transition-all duration-300 ease-in-out hover:shadow-md group"
      >
        <div class="flex items-center space-x-2 px-3 py-2">
          <PaperclipIcon size="16" class="text-gray-500 group-hover:text-primary" />
          <Tooltip>
            <TooltipTrigger as-child>
              <div
                class="max-w-[12rem] overflow-hidden text-ellipsis whitespace-nowrap text-sm font-medium text-primary group-hover:text-gray-900"
              >
                {{ getAttachmentName(attachment.filename) }}
              </div>
            </TooltipTrigger>
            <TooltipContent>
              <p class="text-sm">{{ attachment.filename }}</p>
            </TooltipContent>
          </Tooltip>
          <span class="text-xs text-gray-500">
            {{ formatBytes(attachment.size) }}
          </span>
        </div>
        <button
          @click.stop="onDelete(attachment.uuid)"
          class="p-2 text-gray-400 hover:text-red-500 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-opacity-50 rounded transition-colors duration-300 ease-in-out"
          title="Remove attachment"
        >
          <X size="14" />
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup>
import { formatBytes } from '@/utils/file.js'
import { X, Paperclip as PaperclipIcon } from 'lucide-vue-next'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

defineProps({
  attachments: {
    type: Array,
    required: true
  },
  onDelete: {
    type: Function,
    required: true
  }
})

const getAttachmentName = (name) => {
  return name.length > 20 ? name.substring(0, 17) + '...' : name
}
</script>

<style scoped>
.attachment-list-move,
.attachment-list-enter-active,
.attachment-list-leave-active {
  transition: all 0.5s ease;
}

.attachment-list-enter-from,
.attachment-list-leave-to {
  opacity: 0;
  transform: translateX(30px);
}

.attachment-list-leave-active {
  position: absolute;
}
</style>

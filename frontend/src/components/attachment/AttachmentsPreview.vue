<template>
  <div
    class="flex m-2 items-end text-sm overflow-hidden text-ellipsis whitespace-nowrap cursor-pointer"
  >
    <div
      v-for="attachment in attachments"
      :key="attachment.uuid"
      class="flex items-center p-1 bg-[#F5F5F4] gap-1 rounded-md max-w-[15rem]"
    >
      <!-- Filename tooltip -->
      <Tooltip>
        <TooltipTrigger as-child>
          <div class="overflow-hidden text-ellipsis whitespace-nowrap">
            {{ getAttachmentName(attachment.filename) }}
          </div>
        </TooltipTrigger>
        <TooltipContent>
          {{ attachment.filename }}
        </TooltipContent>
      </Tooltip>
      <div>
        {{ formatBytes(attachment.size) }}
      </div>
      <div @click="onDelete(attachment.uuid)">
        <X size="13" />
      </div>
    </div>
  </div>
</template>

<script setup>
import { formatBytes } from '@/utils/file.js'

import { X } from 'lucide-vue-next'
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
  return name.substring(0, 20)
}
</script>

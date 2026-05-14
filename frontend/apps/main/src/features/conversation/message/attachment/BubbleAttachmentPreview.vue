<template>
  <div class="flex flex-row flex-wrap gap-2 break-all">
    <div
      v-for="attachment in attachments"
      :key="attachment.uuid"
      class="flex items-center cursor-pointer"
    >
      <div>
        <div
          v-if="isAudio(attachment)"
          class="flex items-center gap-2 rounded border bg-muted/40 px-3 py-2"
        >
          <audio controls preload="auto" class="h-8 max-w-[260px]">
            <source :src="attachment.url" />
          </audio>
          <DownloadLink :url="attachment.url" class="shrink-0" />
        </div>
        <BubbleAttachmentItem v-else :attachment="attachment" @preview="openLightbox" />
      </div>
    </div>
  </div>

  <ImageLightbox
    v-model="lightboxOpen"
    :images="imageAttachments"
    :start-index="lightboxIndex"
  />
</template>

<script setup>
import { ref, computed } from 'vue'
import BubbleAttachmentItem from '@/features/conversation/message/attachment/BubbleAttachmentItem.vue'
import ImageLightbox from '@/components/ImageLightbox.vue'
import DownloadLink from '@/components/DownloadLink.vue'

const props = defineProps({
  attachments: { type: Array, required: true }
})

const isImage = (attachment) => (attachment.content_type || '').startsWith('image/')
const isAudio = (attachment) => (attachment.content_type || '').startsWith('audio/')

const imageAttachments = computed(() =>
  (props.attachments || []).filter(isImage)
)

const lightboxOpen = ref(false)
const lightboxIndex = ref(0)

const openLightbox = (attachment) => {
  const idx = imageAttachments.value.findIndex((a) => a.uuid === attachment.uuid)
  lightboxIndex.value = idx >= 0 ? idx : 0
  lightboxOpen.value = true
}
</script>

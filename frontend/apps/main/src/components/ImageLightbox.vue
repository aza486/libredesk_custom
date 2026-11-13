<template>
  <Dialog :open="modelValue" @update:open="emit('update:modelValue', $event)">
    <DialogPortal>
      <!-- Use the same layer/focus stack as the composer. The image viewer must
           stay inside this layer instead of teleporting outside its focus trap. -->
      <DialogContent
        class="fixed inset-0 z-[100] outline-none"
        :aria-describedby="undefined"
        @keydown.capture="onKeydown"
      >
        <DialogTitle class="sr-only">{{ t('globals.terms.image', 2) }}</DialogTitle>
        <VueEasyLightbox
          :visible="viewerVisible"
          :imgs="imgs"
          :index="index"
          :loop="images.length > 1"
          esc-disabled
          @vue:mounted="viewerVisible = true"
          @hide="close"
          @on-index-change="onIndexChange"
        />
      </DialogContent>
    </DialogPortal>
  </Dialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import VueEasyLightbox from 'vue-easy-lightbox'
import { DialogContent, DialogPortal } from 'radix-vue'
import { Dialog, DialogTitle } from '@shared-ui/components/ui/dialog'
import { useI18n } from 'vue-i18n'

const props = defineProps({
  modelValue: { type: Boolean, required: true },
  images: { type: Array, required: true },
  startIndex: { type: Number, default: 0 }
})
const emit = defineEmits(['update:modelValue'])
const { t } = useI18n()

const index = ref(0)
// Dialog content mounts on demand. Give the viewer its normal false -> true
// transition so it initializes images and saves/restores the existing body lock.
const viewerVisible = ref(false)

const imgs = computed(() =>
  props.images.map((img) => ({ src: img.url, title: img.name || '' }))
)

function clamp(n, min, max) {
  return Math.min(Math.max(n, min), max)
}

function close() {
  emit('update:modelValue', false)
}

function onIndexChange(_prev, next) {
  index.value = next
}

function step(delta) {
  const total = props.images.length
  if (total <= 1) return
  index.value = (index.value + delta + total) % total
}

const KEY_ACTIONS = {
  ArrowLeft: () => step(-1),
  ArrowRight: () => step(1),
  ArrowUp: () => {},
  ArrowDown: () => {},
  PageUp: () => {},
  PageDown: () => {}
}

// Handle navigation only inside the active focus scope. Escape belongs to Radix,
// which dismisses only the topmost layer. Stop navigation before the viewer's
// document listener can apply the same step a second time.
function onKeydown(e) {
  if (!props.modelValue) return
  const action = KEY_ACTIONS[e.key]
  if (!action) return
  e.preventDefault()
  e.stopPropagation()
  action()
}

watch(
  () => props.modelValue,
  (open) => {
    viewerVisible.value = false
    if (open) {
      index.value = clamp(props.startIndex, 0, props.images.length - 1)
    }
  },
  { immediate: true }
)
</script>

<style>
.vel-img-wrapper,
.vel-img,
.vel-fade-enter-active,
.vel-fade-leave-active {
  transition: none !important;
}
</style>

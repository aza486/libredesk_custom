<template>
  <div ref="container" class="relative flex min-w-0 items-center" data-conversation-tags>
    <!-- Reserve 8px on each side of the 1px divider as one measured footprint. -->
    <div
      v-if="separator && hasContent"
      class="flex w-[17px] shrink-0 items-center justify-center"
      aria-hidden="true"
      data-tag-separator
    >
      <span class="h-4 w-px bg-border" />
    </div>
    <div
      ref="visibleRow"
      v-show="hasContent"
      class="flex min-w-0 flex-1 items-center gap-1 overflow-hidden"
      data-tag-row
    >
      <span
        v-for="(tag, index) in visibleTags"
        :key="index"
        :class="[tagClass, borderClass(tag)]"
        data-visible-tag
      >
        {{ tag }}
      </span>
      <Tooltip v-if="hasContent && hiddenTags.length">
        <TooltipTrigger asChild>
          <span
            class="inline-flex h-6 shrink-0 items-center rounded-md px-1 text-[11px] font-medium leading-4 text-muted-foreground focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring"
            tabindex="0"
            :aria-label="hiddenTags.join(', ')"
            data-tag-overflow
            @click.prevent.stop
            @keydown.enter.prevent.stop
            @keydown.space.prevent.stop
            >…</span
          >
        </TooltipTrigger>
        <TooltipContent class="max-w-72 break-words">
          <ul>
            <li v-for="(tag, index) in hiddenTags" :key="index">{{ tag }}</li>
          </ul>
        </TooltipContent>
      </Tooltip>
    </div>
    <!-- Intrinsic widths remain measurable even while their visible copies are hidden. -->
    <div class="pointer-events-none invisible absolute inset-0 overflow-hidden" aria-hidden="true">
      <div ref="measurement" class="flex w-max items-center gap-1">
        <span v-if="separator" class="w-[17px] shrink-0" data-separator-measure />
        <span
          v-for="(tag, index) in orderedTags"
          :key="index"
          :class="[tagClass, borderClass(tag)]"
          data-tag-measure
          >{{ tag }}</span
        >
        <span
          class="inline-flex h-6 shrink-0 items-center rounded-md px-1 text-[11px] font-medium leading-4"
          data-overflow-measure
          >…</span
        >
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { Tooltip, TooltipContent, TooltipTrigger } from '@shared-ui/components/ui/tooltip'
import { visibleTagCount } from './tag-overflow'
import { prioritizeConversationTags, warningTagKind } from './warning-tags'

const props = defineProps({
  tags: { type: Array, default: () => [] },
  separator: { type: Boolean, default: false }
})
const tagClass =
  'inline-flex h-6 shrink-0 items-center whitespace-nowrap rounded-md border bg-muted px-[5px] text-[11px] font-medium leading-4 text-muted-foreground'
const borderClass = (tag) => {
  const kind = warningTagKind(tag)
  return kind === 'error'
    ? 'border-red-500'
    : kind === 'human'
      ? 'border-yellow-500'
      : 'border-transparent'
}
const container = ref(null)
const visibleRow = ref(null)
const measurement = ref(null)
const count = ref(0)
const hasContent = ref(false)
const orderedTags = computed(() => prioritizeConversationTags(props.tags))
const visibleTags = computed(() => orderedTags.value.slice(0, count.value))
const hiddenTags = computed(() => orderedTags.value.slice(count.value))
let observer
let frame = null
let dirty = true
let widths = []
let indicator = 0
let gap = 0
let separatorSpace = 0
let stopped = false

function update() {
  frame = null
  if (!container.value || !measurement.value) return
  if (dirty) {
    widths = [...measurement.value.querySelectorAll('[data-tag-measure]')].map(
      (el) => el.getBoundingClientRect().width
    )
    indicator = measurement.value
      .querySelector('[data-overflow-measure]')
      .getBoundingClientRect().width
    gap = parseFloat(getComputedStyle(visibleRow.value).columnGap) || 0
    separatorSpace = props.separator
      ? measurement.value.querySelector('[data-separator-measure]').getBoundingClientRect().width
      : 0
    dirty = false
  }
  // Capacity comes from the stable outer allocation, not the conditional
  // separator/row. Showing or hiding them therefore cannot cause a resize loop.
  const available = Math.max(0, container.value.getBoundingClientRect().width - separatorSpace)
  const fitted = visibleTagCount(widths, available, indicator, gap)
  hasContent.value = widths.length > 0 && (fitted > 0 || available >= indicator)
  count.value = hasContent.value ? fitted : 0
}

function schedule(remeasure = false) {
  if (stopped) return
  dirty ||= remeasure
  if (frame === null) frame = requestAnimationFrame(update)
}

onMounted(() => {
  observer = new ResizeObserver((entries) => {
    schedule(entries.some((entry) => entry.target === measurement.value))
  })
  observer.observe(container.value)
  observer.observe(measurement.value)
  schedule(true)
})

watch(
  () => [props.tags, props.separator],
  async () => {
    await nextTick()
    schedule(true)
  },
  { deep: true }
)

onBeforeUnmount(() => {
  stopped = true
  observer?.disconnect()
  if (frame !== null) cancelAnimationFrame(frame)
})
</script>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { getTextFromHTML } from '@shared-ui/utils/string'
import api from '@main/api'
import AIActionPanel from './AIActionPanel.vue'
import VoiceInput from '@main/components/voice/VoiceInput.vue'
import { Textarea } from '@shared-ui/components/ui/textarea'
import { Button } from '@shared-ui/components/ui/button'
const props = defineProps({
  open: Boolean,
  embedded: Boolean,
  conversationUUID: { type: String, default: '' },
  draftHTML: { type: String, default: '' },
  draftText: { type: String, default: '' },
  editorMode: { type: String, default: '' }
})
const emit = defineEmits(['update:open', 'adopt'])
const instructions = ref(''),
  mode = ref('generate'),
  result = ref(''),
  busy = ref(false)
const error = ref(''),
  voiceBusy = ref(false),
  changed = ref(false)
const hasDraft = computed(() => !!getTextFromHTML(props.draftHTML).trim())
const tooLong = computed(
  () =>
    [...instructions.value].length > 8000 ||
    (Boolean(instructions.value.trim()) &&
      mode.value === 'edit' &&
      [...props.draftHTML].length > 64000)
)
let revision = 0,
  controller,
  snapshot
function cancel() {
  revision++
  controller?.abort()
  busy.value = false
}
watch(
  () => [props.open, props.conversationUUID],
  () => {
    cancel()
    result.value = ''
    instructions.value = ''
    error.value = ''
    changed.value = false
    mode.value = hasDraft.value ? 'edit' : 'generate'
  },
  { immediate: true }
)
watch(
  () => [props.draftHTML, props.editorMode],
  () => {
    if (snapshot && (busy.value || result.value)) changed.value = true
  }
)
watch([instructions, mode], () => {
  cancel()
  result.value = ''
  changed.value = false
  error.value = ''
})
onBeforeUnmount(cancel)
async function generate() {
  if (busy.value || voiceBusy.value || tooLong.value || !props.conversationUUID) return
  const run = ++revision
  controller = new AbortController()
  snapshot = { uuid: props.conversationUUID, html: props.draftHTML, editorMode: props.editorMode }
  result.value = ''
  error.value = ''
  changed.value = false
  busy.value = true
  // Empty dialog instructions preserve the legacy request, including the existing
  // editor text as instruction. No new mode or draft parameters in this case.
  const payload = { conversation_uuid: snapshot.uuid, instruction: props.draftText }
  if (instructions.value.trim()) {
    payload.instruction = instructions.value
    payload.mode = hasDraft.value ? mode.value : 'generate'
    if (payload.mode === 'edit') payload.draft_html = snapshot.html
  }
  try {
    const response = await api.aiGenerateReply(payload, controller.signal)
    if (run !== revision || !props.open || snapshot.uuid !== props.conversationUUID) return
    const html = response?.data?.data
    if (typeof html !== 'string' || !getTextFromHTML(html).trim()) throw new Error('empty')
    result.value = html
  } catch {
    if (run === revision && props.open) error.value = 'failed'
  } finally {
    if (run === revision) busy.value = false
  }
}
function adopt() {
  if (
    !result.value ||
    busy.value ||
    changed.value ||
    snapshot?.uuid !== props.conversationUUID ||
    snapshot.html !== props.draftHTML ||
    snapshot.editorMode !== props.editorMode
  )
    return
  emit('adopt', { resultHTML: result.value, ...snapshot })
  emit('update:open', false)
}
</script>
<template>
  <AIActionPanel
    :open="open"
    :embedded="embedded"
    :title="$t('aiGenerate.title')"
    :description="$t('aiGenerate.description')"
    content-class="sm:max-w-2xl"
    @update:open="emit('update:open', $event)"
  >
    <div v-if="hasDraft" class="space-y-2">
      <p class="text-sm text-muted-foreground">{{ $t('aiGenerate.draftHint') }}</p>
      <label class="flex items-center gap-2 text-sm"
        ><input v-model="mode" type="radio" value="generate" :disabled="busy" />{{
          $t('aiGenerate.new')
        }}</label
      >
      <label class="flex items-center gap-2 text-sm"
        ><input v-model="mode" type="radio" value="edit" :disabled="busy" />{{
          $t('aiGenerate.edit')
        }}</label
      >
    </div>
    <label for="ai-generate-instructions" class="text-sm font-medium">{{
      $t('aiGenerate.instructions')
    }}</label>
    <Textarea
      id="ai-generate-instructions"
      v-model="instructions"
      rows="4"
      :disabled="busy"
      :aria-invalid="tooLong"
    />
    <VoiceInput
      v-model="instructions"
      :active="open"
      :context-key="conversationUUID"
      :disabled="busy"
      :max-chars="8000"
      @busy="voiceBusy = $event"
    />
    <p v-if="tooLong" role="alert" class="text-sm text-destructive">{{ $t('aiGenerate.limit') }}</p>
    <Button :disabled="busy || voiceBusy || tooLong" @click="generate">{{
      $t(
        busy
          ? 'aiGenerate.processing'
          : hasDraft && mode === 'edit' && instructions.trim()
            ? 'aiGenerate.apply'
            : 'aiGenerate.create'
      )
    }}</Button>
    <div
      v-if="result"
      class="max-h-64 overflow-auto whitespace-pre-wrap break-words rounded-md border bg-muted p-3 text-sm"
      :aria-label="$t('aiGenerate.preview')"
      role="status"
    >
      {{ getTextFromHTML(result) }}
    </div>
    <p v-if="changed" role="alert" class="text-sm text-destructive">
      {{ $t('aiGenerate.changed') }}
    </p>
    <p v-if="error" role="alert" class="text-sm text-destructive">{{ $t('aiGenerate.failed') }}</p>
    <template #footer>
      <Button variant="outline" @click="emit('update:open', false)">{{
        $t('aiGenerate.discard')
      }}</Button>
      <Button :disabled="busy || voiceBusy || changed || !result" @click="adopt">{{
        $t('aiGenerate.adopt')
      }}</Button>
    </template>
  </AIActionPanel>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import api from '@main/api'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue
} from '@shared-ui/components/ui/select'
import { Button } from '@shared-ui/components/ui/button'
import { Alert, AlertDescription } from '@shared-ui/components/ui/alert'
const props = defineProps({
  conversationUuid: { type: String, required: true },
  modelValue: { type: String, default: '' },
  lockedSource: { type: String, default: '' },
  disabled: Boolean
})
const emit = defineEmits(['update:modelValue'])
const sources = ref([]),
  busy = ref(false),
  error = ref(''),
  hasMore = ref(false),
  total = ref(0)
let page = 0,
  generation = 0
const selected = computed({
  get: () => props.lockedSource || props.modelValue || '__none',
  set: (v) => emit('update:modelValue', v === '__none' ? '' : v)
})
async function load(reset = false) {
  if (busy.value && !reset) return
  const run = reset ? ++generation : generation
  if (reset) {
    sources.value = []
    page = 0
    total.value = 0
    hasMore.value = false
  }
  busy.value = true
  error.value = ''
  try {
    const data = (
      await api.getReplySources(props.conversationUuid, { page: page + 1, page_size: 100 })
    )?.data?.data
    if (run !== generation) return
    if (!Array.isArray(data?.results)) throw new Error('invalid reply sources')
    sources.value.push(...data.results)
    page++
    total.value = data.total
    hasMore.value = data.has_more
    // A single source is unambiguous; two or more never imply "latest".
    if (total.value === 1 && !props.modelValue && !props.lockedSource)
      emit('update:modelValue', sources.value[0].uuid)
  } catch {
    if (run === generation)
      error.value = 'Kundenmails konnten nicht geladen werden. Bitte erneut versuchen.'
  } finally {
    if (run === generation) busy.value = false
  }
}
watch(
  () => props.conversationUuid,
  () => load(true),
  { immediate: true }
)
onBeforeUnmount(() => generation++)
</script>
<template>
  <div class="mb-2 space-y-2 rounded-md border p-2 text-sm">
    <p class="font-medium">Antwort auf Kundenmail</p>
    <Select v-model="selected" :disabled="disabled || busy || !!lockedSource">
      <SelectTrigger aria-label="Ursprüngliche Kundenmail auswählen"
        ><SelectValue placeholder="Kundenmail auswählen"
      /></SelectTrigger>
      <SelectContent>
        <SelectItem value="__none">Keine Quellzuordnung</SelectItem>
        <SelectItem
          v-if="
            (lockedSource || modelValue) &&
            !sources.some((s) => s.uuid === (lockedSource || modelValue))
          "
          :value="lockedSource || modelValue"
          >{{ lockedSource || modelValue }}</SelectItem
        >
        <SelectItem v-for="source in sources" :key="source.uuid" :value="source.uuid"
          >{{ source.created_at }} · {{ source.preview || source.uuid }}</SelectItem
        >
      </SelectContent>
    </Select>
    <p v-if="lockedSource" class="text-xs text-muted-foreground">
      Die Quelle ist durch die KI-/Feedback-Aktion festgelegt.
    </p>
    <p v-else-if="!modelValue" class="text-xs text-muted-foreground">
      {{
        total > 1
          ? 'Mehrere Kundenmails: Bitte die beantwortete Nachricht auswählen.'
          : 'Keine eindeutige Quellzuordnung vorhanden.'
      }}
      Ohne Zuordnung bleibt der automatische Versandabschluss ausstehend. Der E-Mail-Versand bleibt
      möglich.
    </p>
    <Button v-if="hasMore" size="sm" variant="ghost" :disabled="busy" @click="load()"
      >Weitere Kundenmails laden</Button
    >
    <Alert v-if="error" variant="destructive"
      ><AlertDescription
        >{{ error }}
        <Button size="sm" variant="outline" :disabled="busy" @click="load(page === 0)"
          >Erneut laden</Button
        ></AlertDescription
      ></Alert
    >
  </div>
</template>

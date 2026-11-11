<script setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import api from '@main/api'
import { useEmitter } from '@main/composables/useEmitter'
import { EMITTER_EVENTS } from '@main/constants/emitterEvents'
import { useConversationStore } from '@main/stores/conversation'
import { handleHTTPError } from '@shared-ui/utils/http'
import { Input } from '@shared-ui/components/ui/input'
import { Button } from '@shared-ui/components/ui/button'

const emit = defineEmits(['cancel'])
const { t } = useI18n()
const emitter = useEmitter()
const store = useConversationStore()
const addresses = ref([])
const address = ref('')
const pendingRemoval = ref('')
const busy = ref(false)
const loading = ref(true)
const error = ref('')
let disposed = false
let loadVersion = 0
async function load() {
  const version = ++loadVersion
  try {
    const response = await api.getServiceEmailAddresses()
    if (!disposed && version === loadVersion) {
      addresses.value = response.data.data || []
      error.value = ''
    }
  } catch (e) {
    if (!disposed && version === loadVersion) error.value = handleHTTPError(e).message
  } finally {
    if (!disposed && version === loadVersion) loading.value = false
  }
}
function cancel() { address.value = ''; emit('cancel') }
async function save(remove = false) {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    if (remove) await api.removeServiceEmailAddress(pendingRemoval.value)
    else await api.addServiceEmailAddress(address.value.trim())
    address.value = ''
    pendingRemoval.value = ''
    emitter.emit('service-emails-changed')
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'inbox-list' })
    store.refreshConversationListForRealtimeEvent()
  } catch (e) {
    error.value = handleHTTPError(e).message
  } finally {
    busy.value = false
  }
}
onMounted(() => {
  load()
  emitter.on('service-emails-changed', load)
})
onUnmounted(() => {
  disposed = true
  emitter.off('service-emails-changed', load)
})
</script>

<template>
  <div class="space-y-4">
    <p class="text-sm text-muted-foreground">{{ t('serviceEmails.description') }}</p>
    <form class="flex flex-wrap gap-2" @submit.prevent="save()">
      <label class="min-w-0 flex-1 text-sm">
        {{ t('serviceEmails.email') }}
        <Input v-model="address" type="email" required :disabled="busy" autocomplete="off" />
      </label>
      <Button class="self-end" :disabled="busy || loading" type="submit">{{
        t('serviceEmails.add')
      }}</Button>
      <Button
        class="self-end"
        variant="outline"
        type="button"
        :disabled="busy"
        @click="cancel"
        >{{ t('serviceEmails.cancel') }}</Button
      >
    </form>
    <p v-if="error" role="alert" class="text-sm text-destructive">{{ error }}</p>
    <p v-if="loading" role="status" class="text-sm text-muted-foreground">
      {{ t('serviceEmails.loading') }}
    </p>
    <div v-else-if="addresses.length" class="max-h-72 overflow-y-auto divide-y rounded-md border">
      <div
        v-for="entry in addresses"
        :key="entry"
        class="flex items-center justify-between gap-2 px-3 py-2 text-sm"
      >
        <span class="min-w-0 break-all">{{ entry }}</span>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          :disabled="busy"
          :aria-label="t('serviceEmails.removeAddress', { address: entry })"
          @click="pendingRemoval = entry"
          >{{ t('serviceEmails.remove') }}</Button
        >
      </div>
    </div>
    <p v-else class="text-sm text-muted-foreground">{{ t('serviceEmails.empty') }}</p>
    <div v-if="pendingRemoval" role="alert" class="space-y-3 rounded-md border p-3 text-sm">
      <p>{{ t('serviceEmails.confirmRemove', { address: pendingRemoval }) }}</p>
      <div class="flex gap-2">
        <Button variant="destructive" type="button" :disabled="busy" @click="save(true)">{{
          t('serviceEmails.remove')
        }}</Button>
        <Button variant="outline" type="button" :disabled="busy" @click="pendingRemoval = ''">{{
          t('serviceEmails.cancel')
        }}</Button>
      </div>
    </div>
  </div>
</template>

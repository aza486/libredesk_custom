<script setup>
import { computed, ref } from 'vue'
import { Undo2 } from 'lucide-vue-next'
import { useI18n } from 'vue-i18n'
import { useConversationStore } from '@/stores/conversation'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http'
import api from '@/api'

const props = defineProps({ close: Function, beforeReturn: Function })
const store = useConversationStore()
const emitter = useEmitter()
const { t } = useI18n()
const busy = ref(false)
const available = computed(
  () =>
    store.current?.can_return_to_support === true &&
    store.current?.customer_ticket === true &&
    store.current?.customer_redistributed === true &&
    !(store.current.tags || []).includes('🗑Spam')
)

async function returnToSupport() {
  if (busy.value || !available.value) return
  busy.value = true
  const uuid = store.current.uuid
  try {
    await props.beforeReturn?.()
    await api.returnCustomerToSupport(uuid)
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message || t('conversation.routing.returnFailed')
    })
    busy.value = false
    return
  }
  props.close?.()
  emitter.emit(EMITTER_EVENTS.SHOW_TOAST, { description: t('conversation.routing.returnSuccess') })
  emitter.emit(EMITTER_EVENTS.REFRESH_LIST, { model: 'inbox-list' })
  store.refreshConversationListForRealtimeEvent()
  // The return has committed. A subsequent read failure must not report a
  // failed return or repeat the mutation; normal realtime refresh can recover.
  try {
    const response = await api.getConversation(uuid)
    if (store.current?.uuid === uuid) Object.assign(store.current, response.data.data)
  } catch (error) {
    console.warn('Could not refresh returned conversation', error)
  } finally {
    busy.value = false
  }
}
</script>

<template>
  <div v-if="available" class="border-t p-1">
    <button
      type="button"
      class="flex w-full items-center gap-2 rounded-sm px-2 py-2 text-left text-sm hover:bg-accent focus-visible:outline focus-visible:outline-2 disabled:opacity-50"
      :disabled="busy"
      @click="returnToSupport"
    >
      <Undo2 class="size-4 shrink-0" />
      <span>{{ t('conversation.routing.returnToSupport') }}</span>
    </button>
  </div>
</template>

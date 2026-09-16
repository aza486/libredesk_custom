<template>
  <AdminSplitLayout>
    <template #content>
      <div class="space-y-5">
        <div>
          <h2 class="text-lg font-semibold">Service-Mails</h2>
          <p class="text-sm text-muted-foreground">
            E-Mails dieser Absender werden nie automatisch durch KI beantwortet.
          </p>
        </div>
        <form class="flex gap-2" @submit.prevent="addAddress">
          <Input v-model="newAddress" type="email" placeholder="service@example.com" required />
          <Button :disabled="saving">Hinzufügen</Button>
        </form>
        <div v-if="addresses.length" class="divide-y rounded-md border">
          <div
            v-for="address in addresses"
            :key="address"
            class="flex items-center justify-between px-3 py-2 text-sm"
          >
            <span>{{ address }}</span>
            <Button variant="ghost" size="sm" @click="removeAddress(address)">Entfernen</Button>
          </div>
        </div>
        <p v-else class="text-sm text-muted-foreground">
          Noch keine Service-Mail-Adressen hinterlegt.
        </p>
      </div>
    </template>
    <template #help
      >Hinterlege automatische Absender wie Rechnungs-, Monitoring- oder Systemdienste.</template
    >
  </AdminSplitLayout>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import AdminSplitLayout from '@/layouts/admin/AdminSplitLayout.vue'
import { Button } from '@shared-ui/components/ui/button'
import { Input } from '@shared-ui/components/ui/input'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents'
import { handleHTTPError } from '@shared-ui/utils/http.js'
import api from '@/api'

const addresses = ref([])
const newAddress = ref('')
const saving = ref(false)
const emitter = useEmitter()

const load = async () => {
  try {
    addresses.value = (await api.getServiceEmailAddresses()).data.data || []
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
const addAddress = async () => {
  if (!newAddress.value || saving.value) return
  saving.value = true
  try {
    await api.addServiceEmailAddress(newAddress.value)
    newAddress.value = ''
    await load()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    saving.value = false
  }
}
const removeAddress = async (address) => {
  try {
    await api.removeServiceEmailAddress(address)
    await load()
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}
onMounted(load)
</script>

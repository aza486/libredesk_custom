<template>
  <Spinner v-if="isLoading" />
  <AdminPageWithHelp>
    <template #content>
      <template v-if="router.currentRoute.value.path === '/admin/templates'">
        <div :class="{ 'opacity-50 transition-opacity duration-300': isLoading }">
          <div class="flex justify-between mb-5">
            <div></div>
            <div class="flex justify-end mb-4">
              <Button @click="navigateToNewTemplate" :disabled="templateType !== 'email_outgoing'">
                New template
              </Button>
            </div>
          </div>
          <div>
            <Tabs default-value="email_outgoing" v-model="templateType">
              <TabsList class="grid w-full grid-cols-2 mb-5">
                <TabsTrigger value="email_outgoing">Outgoing email templates</TabsTrigger>
                <TabsTrigger value="email_notification">Email notification templates</TabsTrigger>
              </TabsList>
              <TabsContent value="email_outgoing">
                <DataTable :columns="outgoingEmailTemplatesColumns" :data="templates" />
              </TabsContent>
              <TabsContent value="email_notification">
                <DataTable :columns="emailNotificationTemplates" :data="templates" />
              </TabsContent>
            </Tabs>
          </div>
        </div>
      </template>
      <template v-else>
        <router-view />
      </template>
    </template>

    <template #help>
      <p>Design templates for customer communications and responses.</p>
      <p>Configure internal team notification templates.</p>
    </template>
  </AdminPageWithHelp>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import DataTable from '@/components/datatable/DataTable.vue'
import {
  emailNotificationTemplates,
  outgoingEmailTemplatesColumns
} from '@/features/admin/templates/dataTableColumns.js'
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { useStorage } from '@vueuse/core'
import AdminPageWithHelp from '@/layouts/admin/AdminPageWithHelp.vue'
import api from '@/api'

const templateType = useStorage('templateType', 'email_outgoing')
const templates = ref([])
const isLoading = ref(false)
const router = useRouter()
const emit = useEmitter()

onMounted(async () => {
  fetchAll()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const fetchAll = async () => {
  try {
    isLoading.value = true
    const resp = await api.getTemplates(templateType.value)
    templates.value = resp.data.data
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: error.message
    })
  } finally {
    isLoading.value = false
  }
}

const refreshList = (data) => {
  if (data?.model === 'templates') fetchAll()
}

const navigateToNewTemplate = () => {
  router.push({
    name: 'new-template',
    query: { type: templateType.value }
  })
}

watch(
  templateType,
  () => {
    templates.value = []
    fetchAll()
  },
  { immediate: true }
)
</script>

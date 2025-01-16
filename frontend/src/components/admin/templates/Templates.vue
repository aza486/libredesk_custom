<template>
    <template v-if="router.currentRoute.value.path === '/admin/templates'">
      <div class="flex justify-between mb-5">
        <div></div>
        <div class="flex justify-end mb-4">
          <Button @click="navigateToAddTemplate"> New template </Button>
        </div>
      </div>
      <div>
        <Spinner v-if="isLoading"></Spinner>
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
    </template>
    <template v-else>
      <router-view/>
    </template>
</template>

<script setup>
import { ref, onMounted, onUnmounted, watch } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { emailNotificationTemplates, outgoingEmailTemplatesColumns } from '@/components/admin/templates/dataTableColumns.js'
import { Button } from '@/components/ui/button'

import { useRouter } from 'vue-router'
import { Spinner } from '@/components/ui/spinner'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { useStorage } from '@vueuse/core'
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
  } finally {
    isLoading.value = false
  }
}

const refreshList = (data) => {
  if (data?.model === 'templates') fetchAll()
}

const navigateToAddTemplate = () => {
  router.push('/admin/templates/new')
}

watch(templateType, () => {
  fetchAll()
})
</script>

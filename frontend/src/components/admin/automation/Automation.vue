<template>
    <div v-if="router.currentRoute.value.path === '/admin/automations'">
      <div class="flex justify-between mb-5">
        <div class="ml-auto">
          <Button @click="newRule">New rule</Button>
        </div>
      </div>
      <div v-if="selectedTab">
        <AutomationTabs v-model="selectedTab" />
      </div>
    </div>
    <router-view />
</template>

<script setup>
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import { useStorage } from '@vueuse/core'
import AutomationTabs from '@/components/admin/automation/AutomationTabs.vue'

const router = useRouter()
const selectedTab = useStorage('automationsTab', 'new_conversation')
const newRule = () => {
  router.push({ path: `/admin/automations/new`, query: { type: selectedTab.value } })
}
</script>
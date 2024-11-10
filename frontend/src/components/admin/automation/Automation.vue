<template>
  <div class="flex justify-between mb-5">
    <PageHeader title="Automations" description="Manage automations and time triggers" />
    <div>
      <Button size="sm" @click="newRule">New rule</Button>
    </div>
  </div>
  <div>
    <AutomationTabs v-model="selectedTab" />
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { Button } from '@/components/ui/button'
import AutomationTabs from '@/components/admin/automation/AutomationTabs.vue'
import { useRouter } from 'vue-router'
import PageHeader from '../common/PageHeader.vue'

const router = useRouter()
const selectedTab = ref('new_conversation')

const newRule = () => {
  router.push({ path: `/admin/automations/new`, query: { type: selectedTab.value } })
}
</script>
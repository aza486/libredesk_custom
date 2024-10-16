<template>
  <div class="flex justify-between mb-5">
    <PageHeader title="Automations" description="Manage automations and time triggers" />
    <div>
      <Button size="sm" @click="newRule">New rule</Button>
    </div>
  </div>
  <div>
    <AutomationTabs />
  </div>
</template>

<script setup>
import { Button } from '@/components/ui/button'
import AutomationTabs from '@/components/admin/automation/AutomationTabs.vue'
import { useRouter } from 'vue-router'
import PageHeader from '../common/PageHeader.vue'
const router = useRouter()

const newRule = () => {
  router.push({ path: `/admin/automations/new` })
}
</script>

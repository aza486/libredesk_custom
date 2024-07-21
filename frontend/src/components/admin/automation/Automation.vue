<template>
  <div class="flex justify-between mb-5">
    <div>
      <span class="admin-title">Automations</span>
      <p class="text-muted-foreground text-sm">Create automations & time triggers.</p>
    </div>
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
const router = useRouter()

const newRule = () => {
  router.push({ path: `/admin/automations/new`})
}
</script>
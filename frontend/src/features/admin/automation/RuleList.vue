<template>
  <div class="flex flex-col box px-5 justify-center py-3">
    <div class="flex justify-between space-y-1">
      <div>
        <span class="sub-title space-x-3 flex justify-center items-center">
          <div class="text-base">
            {{ rule.name }}
          </div>
          <div class="mb-1">
            <Badge v-if="rule.enabled" class="text-[9px]">Enabled</Badge>
            <Badge v-else variant="secondary">Disabled</Badge>
          </div>
        </span>
      </div>
      <div>
        <DropdownMenu>
          <DropdownMenuTrigger as-child>
            <button>
              <EllipsisVertical size="18"></EllipsisVertical>
            </button>
          </DropdownMenuTrigger>
          <DropdownMenuContent>
            <DropdownMenuItem @click="navigateToEditRule(rule.id)">
              <span>Edit</span>
            </DropdownMenuItem>
            <DropdownMenuItem @click="() => (alertOpen = true)">
              <span>Delete</span>
            </DropdownMenuItem>
            <DropdownMenuItem @click="$emit('toggle-rule', rule.id)" v-if="rule.enabled">
              <span>Disable</span>
            </DropdownMenuItem>
            <DropdownMenuItem @click="$emit('toggle-rule', rule.id)" v-else>
              <span>Enable</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </div>
    <p class="text-sm-muted">{{ rule.description }}</p>
  </div>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Delete Rule</AlertDialogTitle>
        <AlertDialogDescription>
          This action cannot be undone. This will permanently delete the automation rule.
        </AlertDialogDescription>
      </AlertDialogHeader>
      <AlertDialogFooter>
        <AlertDialogCancel>Cancel</AlertDialogCancel>
        <AlertDialogAction @click="handleDelete">Delete</AlertDialogAction>
      </AlertDialogFooter>
    </AlertDialogContent>
  </AlertDialog>
</template>

<script setup>
import { ref } from 'vue'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle
} from '@/components/ui/alert-dialog'
import { EllipsisVertical } from 'lucide-vue-next'
import { useRouter } from 'vue-router'
import { Badge } from '@/components/ui/badge'

const router = useRouter()
const alertOpen = ref(false)
const emit = defineEmits(['delete-rule', 'toggle-rule'])

const props = defineProps({
  rule: {
    type: Object,
    required: true
  }
})

const navigateToEditRule = (id) => {
  router.push({ path: `/admin/automations/${id}/edit` })
}

const handleDelete = () => {
  emit('delete-rule', props.rule.id)
  alertOpen.value = false
}
</script>

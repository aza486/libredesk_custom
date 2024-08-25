<script setup>
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import api from '@/api'
import { useEmitter } from '@/composables/useEmitter'

const router = useRouter()
const emit = useEmitter()

const props = defineProps({
  role: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

function edit(id) {
  router.push({ path: `/admin/oidc/${id}/edit` })
}

async function deleteOIDC(id) {
  await api.deleteOIDC(id)
  emit.emit('refresh-list', {
    name: 'inbox'
  })
}
</script>

<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only">Open menu</span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="edit(props.role.id)"> Edit </DropdownMenuItem>
      <DropdownMenuItem @click="deleteOIDC(props.role.id)"> Delete </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

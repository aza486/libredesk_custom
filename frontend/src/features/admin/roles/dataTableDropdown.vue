<template>
  <DropdownMenu>
    <DropdownMenuTrigger as-child>
      <Button variant="ghost" class="w-8 h-8 p-0">
        <span class="sr-only">Open menu</span>
        <MoreHorizontal class="w-4 h-4" />
      </Button>
    </DropdownMenuTrigger>
    <DropdownMenuContent>
      <DropdownMenuItem @click="editRole(props.role.id)">Edit</DropdownMenuItem>
      <DropdownMenuItem
        @click="() => (alertOpen = true)"
        v-if="Roles.includes(props.role.name) === false"
      >
        Delete
      </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>

  <AlertDialog :open="alertOpen" @update:open="alertOpen = $event">
    <AlertDialogContent>
      <AlertDialogHeader>
        <AlertDialogTitle>Delete Role</AlertDialogTitle>
        <AlertDialogDescription>
          This action cannot be undone. This will permanently delete the role.
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
import { MoreHorizontal } from 'lucide-vue-next'
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
import { Button } from '@/components/ui/button'
import { useRouter } from 'vue-router'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { Roles } from '@/constants/user'
import api from '@/api'

const alertOpen = ref(false)
const emit = useEmitter()
const router = useRouter()

const props = defineProps({
  role: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

function editRole(id) {
  router.push({ path: `/admin/teams/roles/${id}/edit` })
}

async function handleDelete() {
  try {
    await api.deleteRole(props.role.id)
    alertOpen.value = false
    emitRefreshTeamList()
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
}

const emitRefreshTeamList = () => {
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'team'
  })
}
</script>

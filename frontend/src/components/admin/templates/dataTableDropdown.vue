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
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import { handleHTTPError } from '@/utils/http'
import api from '@/api'

const router = useRouter()
const emitter = useEmitter()
const props = defineProps({
  template: {
    type: Object,
    required: true,
    default: () => ({
      id: ''
    })
  }
})

const editTemplate = (id) => {
  router.push({ path: `/admin/templates/${id}/edit` })
}

const deleteTemplate = async (id) => {
  try {
    await api.deleteTemplate(id)
    emitter.emit(EMITTER_EVENTS.REFRESH_LIST, {
      model: 'templates'
    })
  } catch (error) {
    emitter.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  }
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
      <DropdownMenuItem @click="editTemplate(props.template.id)"> Edit </DropdownMenuItem>
      <DropdownMenuItem @click="deleteTemplate(props.template.id)"> Delete </DropdownMenuItem>
    </DropdownMenuContent>
  </DropdownMenu>
</template>

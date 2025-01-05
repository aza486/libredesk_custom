<template>
  <Dialog v-model:open="dialogOpen">
    <DropdownMenu>
      <DropdownMenuTrigger as-child>
        <Button variant="ghost" class="w-8 h-8 p-0">
          <span class="sr-only">Open menu</span>
          <MoreHorizontal class="w-4 h-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent>
        <DialogTrigger as-child>
          <DropdownMenuItem> Edit </DropdownMenuItem>
        </DialogTrigger>
        <DropdownMenuItem @click="deleteStatus"
          v-if="CONVERSATION_DEFAULT_STATUSES.includes(props.status.name) === false"> Delete </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
    <DialogContent class="sm:max-w-[425px]">
      <DialogHeader>
        <DialogTitle>Edit status</DialogTitle>
        <DialogDescription> Change the status name. Click save when you're done. </DialogDescription>
      </DialogHeader>
      <StatusForm @submit.prevent="onSubmit">
        <template #footer>
          <DialogFooter class="mt-10">
            <Button type="submit"> Save changes </Button>
          </DialogFooter>
        </template>
      </StatusForm>
    </DialogContent>
  </Dialog>
</template>

<script setup>
import { watch, ref } from 'vue'
import { MoreHorizontal } from 'lucide-vue-next'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger
} from '@/components/ui/dropdown-menu'
import { Button } from '@/components/ui/button'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
import StatusForm from './StatusForm.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog'
import { CONVERSATION_DEFAULT_STATUSES } from '@/constants/conversation.js'
import { useEmitter } from '@/composables/useEmitter'
import { handleHTTPError } from '@/utils/http'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api/index.js'

const dialogOpen = ref(false)
const emit = useEmitter()

const props = defineProps({
  status: {
    type: Object,
    required: true,
  }
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const onSubmit = form.handleSubmit(async (values) => {
  await api.updateStatus(props.status.id, values)
  dialogOpen.value = false
  emitRefreshStatusList()
})

const deleteStatus = async () => {
  try {
    await api.deleteStatus(props.status.id)
  } catch (error) {
    emit.emit(EMITTER_EVENTS.SHOW_TOAST, {
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    dialogOpen.value = false
    emitRefreshStatusList()
  }
}

const emitRefreshStatusList = () => {
  emit.emit(EMITTER_EVENTS.REFRESH_LIST, {
    model: 'status'
  })
}

// Watch for changes in initialValues and update the form.
watch(
  () => props.status,
  (newValues) => {
    form.setValues(newValues)
  },
  { immediate: true, deep: true }
)
</script>

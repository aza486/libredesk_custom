<template>
  
  <div class="w-8/12">
    <div class="flex justify-between mb-5">
      <div class="flex justify-end mb-4 w-full">
        <Dialog v-model:open="dialogOpen">
          <DialogTrigger as-child>
            <Button class="ml-auto">New canned response</Button>
          </DialogTrigger>
          <DialogContent class="sm:max-w-[625px]">
            <DialogHeader>
              <DialogTitle>New canned response</DialogTitle>
              <DialogDescription>Set title and content, click save when you're done. </DialogDescription>
            </DialogHeader>
            <CannedResponsesForm @submit="onSubmit">
              <template #footer>
                <DialogFooter class="mt-7">
                  <Button type="submit">Save Changes</Button>
                </DialogFooter>
              </template>
            </CannedResponsesForm>
          </DialogContent>
        </Dialog>
      </div>
    </div>
    <Spinner v-if="formLoading"></Spinner>
    <div v-else>
      <DataTable :columns="columns" :data="cannedResponses" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from './dataTableColumns.js'
import { Button } from '@/components/ui/button'

import { Spinner } from '@/components/ui/spinner'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog'
import CannedResponsesForm from './CannedResponsesForm.vue'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const formLoading = ref(false)
const cannedResponses = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

onMounted(() => {
  getCannedResponses()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, refreshList)
  form.setValues({
    title: "",
    content: "",
  })
})

onUnmounted(() => {
  emit.off(EMITTER_EVENTS.REFRESH_LIST, refreshList)
})

const refreshList = (data) => {
  if (data?.model === 'canned_responses') getCannedResponses()
}

const getCannedResponses = async () => {
  try {
    formLoading.value = true
    const resp = await api.getCannedResponses()
    cannedResponses.value = resp.data.data
  } finally {
    formLoading.value = false
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    formLoading.value = true
    await api.createCannedResponse(values)
    dialogOpen.value = false
    getCannedResponses()
  } catch (error) {
    console.error('Failed to create canned response:', error)
  } finally {
    formLoading.value = false
  }
})
</script>

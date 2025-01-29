<template>
    <div class="flex justify-between mb-5">
      <div class="flex justify-end mb-4 w-full">
        <Dialog v-model:open="dialogOpen">
          <DialogTrigger as-child>
            <Button class="ml-auto">New Tag</Button>
          </DialogTrigger>
          <DialogContent class="sm:max-w-[425px]">
            <DialogHeader>
              <DialogTitle class="mb-1">Create new tag</DialogTitle>
              <DialogDescription> Set tag name. Click save when you're done. </DialogDescription>
            </DialogHeader>
            <TagsForm @submit.prevent="onSubmit">
              <template #footer>
                <DialogFooter class="mt-10">
                  <Button type="submit"> Save changes </Button>
                </DialogFooter>
              </template>
            </TagsForm>
          </DialogContent>
        </Dialog>
      </div>
    </div>
    <Spinner v-if="isLoading"></Spinner>
    <div v-else>
      <DataTable :columns="columns" :data="tags" />
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { Spinner } from '@/components/ui/spinner'
import { columns } from '@/components/admin/conversation/tags/dataTableColumns.js'
import { Button } from '@/components/ui/button'

import TagsForm from './TagsForm.vue'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger
} from '@/components/ui/dialog'
import { useForm } from 'vee-validate'
import { toTypedSchema } from '@vee-validate/zod'
import { formSchema } from './formSchema.js'
import { useEmitter } from '@/composables/useEmitter'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import api from '@/api'

const isLoading = ref(false)
const tags = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getTags()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'tags') getTags()
  })
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const getTags = async () => {
  isLoading.value = true
  const resp = await api.getTags()
  tags.value = resp.data.data
  isLoading.value = false
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    await api.createTag(values)
    dialogOpen.value = false
    getTags()
  } catch (error) {
    console.error('Failed to create tag:', error)
  }
})
</script>

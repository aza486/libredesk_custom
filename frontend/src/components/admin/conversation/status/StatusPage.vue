<template>
  <div>
    <div class="flex justify-between mb-5">
      <PageHeader title="Status" description="Manage conversation statuses" />
      <div class="flex justify-end mb-4">
        <Dialog v-model:open="dialogOpen">
          <DialogTrigger as-child>
            <Button size="sm">New Status</Button>
          </DialogTrigger>
          <DialogContent class="sm:max-w-[425px]">
            <DialogHeader>
              <DialogTitle>New status</DialogTitle>
              <DialogDescription> Set status name. Click save when you're done. </DialogDescription>
            </DialogHeader>
            <form @submit.prevent="onSubmit">
              <FormField v-slot="{ field }" name="name">
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input type="text" placeholder="Processing" v-bind="field" />
                  </FormControl>
                  <FormDescription></FormDescription>
                  <FormMessage />
                </FormItem>
              </FormField>
              <DialogFooter>
                <Button type="submit" size="sm">Save Changes</Button>
              </DialogFooter>
            </form>
          </DialogContent>
        </Dialog>
      </div>
    </div>

    <Spinner v-if="isLoading"></Spinner>
    <div>
      <DataTable :columns="columns" :data="statuses" />
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import DataTable from '@/components/admin/DataTable.vue'
import { columns } from './dataTableColumns.js'
import { Button } from '@/components/ui/button'
import PageHeader from '@/components/admin/common/PageHeader.vue'
import { Spinner } from '@/components/ui/spinner'
import {
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
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
const statuses = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getStatuses()
  emit.on(EMITTER_EVENTS.REFRESH_LIST, (data) => {
    if (data?.model === 'status') getStatuses()
  })
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const getStatuses = async () => {
  try {
    isLoading.value = true
    const resp = await api.getStatuses()
    statuses.value = resp.data.data
  } finally {
    isLoading.value = false
  }
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    isLoading.value = true
    await api.createStatus(values)
    dialogOpen.value = false
    getStatuses()
  } catch (error) {
    console.error('Failed to create status:', error)
  } finally {
    isLoading.value = false
  }
})
</script>

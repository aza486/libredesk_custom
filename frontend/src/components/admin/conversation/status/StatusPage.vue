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
    <div class="w-full">
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
import api from '@/api'

const statuses = ref([])
const emit = useEmitter()
const dialogOpen = ref(false)

onMounted(() => {
  getStatuses()
  emit.on('refresh-list', (data) => {
    if (data?.name === 'status') getStatuses()
  })
})

const form = useForm({
  validationSchema: toTypedSchema(formSchema)
})

const getStatuses = async () => {
  const resp = await api.getStatuses()
  statuses.value = resp.data.data
}

const onSubmit = form.handleSubmit(async (values) => {
  try {
    await api.createStatus(values)
    dialogOpen.value = false
    getStatuses()
  } catch (error) {
    console.error('Failed to create status:', error)
  }
})
</script>

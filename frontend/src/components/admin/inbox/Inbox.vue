<template>
    <template v-if="router.currentRoute.value.path === '/admin/inboxes'">
      <div class="flex justify-between mb-5">
        <div class="flex justify-end w-full mb-4">
          <Button @click="navigateToAddInbox"> New inbox </Button>
        </div>
      </div>
      <div>
        <Spinner v-if="isLoading"></Spinner>
        <DataTable :columns="columns" :data="data" v-else />
      </div>
    </template>
    <template v-else>
      <router-view/>
    </template>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import { h } from 'vue'
import InboxDataTableDropDown from '@/components/admin/inbox/InboxDataTableDropDown.vue'
import { handleHTTPError } from '@/utils/http'
import { useToast } from '@/components/ui/toast/use-toast'
import { Button } from '@/components/ui/button'
import DataTable from '@/components/admin/DataTable.vue'
import { useRouter } from 'vue-router'

import { format } from 'date-fns'
import { Spinner } from '@/components/ui/spinner'
import api from '@/api'

const { toast } = useToast()
const router = useRouter()

const isLoading = ref(false)
const data = ref([])
const showTable = ref(true)

onMounted(async () => {
  await getInboxes()
})

const getInboxes = async () => {
  try {
    isLoading.value = true
    const response = await api.getInboxes()
    data.value = response.data.data
  } catch (error) {
    toast({
      title: 'Error',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
  } finally {
    isLoading.value = false
  }
}

const navigateToAddInbox = () => {
  showTable.value = false
  router.push('/admin/inboxes/new')
}

// Columns for the data table
const columns = [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, 'Name')
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('name'))
    }
  },
  {
    accessorKey: 'channel',
    header: function () {
      return h('div', { class: 'text-center' }, 'Channel')
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('channel'))
    }
  },
  {
    accessorKey: 'disabled',
    header: () => h('div', { class: 'text-center' }, 'Enabled'),
    cell: ({ row }) => {
      const disabled = row.getValue('disabled')
      return h('div', { class: 'text-center' }, [
        h('input', {
          type: 'checkbox',
          checked: !disabled,
          disabled: true
        })
      ])
    }
  },
  {
    accessorKey: 'updated_at',
    header: function () {
      return h('div', { class: 'text-center' }, 'Updated at')
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, format(row.getValue('updated_at'), 'PPpp'))
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    cell: ({ row }) => {
      const inbox = row.original
      return h(
        'div',
        { class: 'relative' },
        h(InboxDataTableDropDown, {
          inbox,
          onEditInbox: (id) => handleEditInbox(id),
          onDeleteInbox: (id) => handleDeleteInbox(id),
          onToggleInbox: (id) => handleToggleInbox(id)
        })
      )
    }
  }
]

const handleEditInbox = (id) => {
  router.push({ path: `/admin/inboxes/${id}/edit` })
}

const handleDeleteInbox = async (id) => {
  await api.deleteInbox(id)
  getInboxes()
}

const handleToggleInbox = async (id) => {
  await api.toggleInbox(id)
  getInboxes()
}
</script>

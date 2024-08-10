<template>
  <div>
    <div class="flex justify-between mb-5">
      <PageHeader title="Inboxes" description="Manage your inboxes" />
      <div class="flex justify-end mb-4">
        <Button @click="navigateToAddInbox" size="sm"> New inbox </Button>
      </div>
    </div>
    <div class="w-full">
      <DataTable :columns="columns" :data="data" />
    </div>
  </div>
  <div>
    <router-view></router-view>
  </div>
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
import PageHeader from '../common/PageHeader.vue'
import { format } from 'date-fns'
import api from '@/api'

const { toast } = useToast()
const router = useRouter()

const data = ref([])
const showTable = ref(true)

onMounted(async () => {
  await getInboxes()
})

const getInboxes = async () => {
  try {
    const response = await api.getInboxes()
    data.value = response.data.data
  } catch (error) {
    toast({
      title: 'Could not fetch inboxes',
      variant: 'destructive',
      description: handleHTTPError(error).message
    })
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
      return h('div', { class: 'text-center' }, 'Modified at')
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
          onToggleInbox: (id) => handleToggleInbox(id),
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

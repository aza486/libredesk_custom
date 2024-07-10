import { h } from 'vue'
import InboxDataTableDropDown from '@/components/admin/InboxDataTableDropDown.vue'

export const columns = [
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
    accessorKey: 'enabled',
    header: () => h('div', { class: 'text-center' }, 'Enabled'),
    cell: ({ row }) => {
      const enabled = row.getValue('enabled')

      return h('div', { class: 'text-center' }, [
        h('input', {
          type: 'checkbox',
          checked: enabled,
          disabled: true
        })
      ])
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
          inbox
        })
      )
    }
  }
]

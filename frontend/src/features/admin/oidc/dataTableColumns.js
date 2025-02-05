import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import { format } from 'date-fns'

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
    accessorKey: 'provider',
    header: function () {
      return h('div', { class: 'text-center' }, 'Provider')
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('provider'))
    }
  },
  {
    accessorKey: 'enabled',
    header: () => h('div', { class: 'text-center' }, 'Enabled'),
    cell: ({ row }) => {
      const enabled = row.getValue('enabled')
      return h('div', { class: 'text-center' }, enabled ? 'Yes' : 'No')
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
      const role = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          role
        })
      )
    }
  }
]

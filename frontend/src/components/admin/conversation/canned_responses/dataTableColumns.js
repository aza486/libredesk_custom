import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import { format } from 'date-fns'

export const columns = [
  {
    accessorKey: 'title',
    header: function () {
      return h('div', { class: 'text-center' }, 'Title')
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('title'))
    }
  },
  {
    accessorKey: 'created_at',
    header: function () {
      return h('div', { class: 'text-center' }, 'Created at')
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, format(row.getValue('created_at'), 'PPpp'))
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
      const canned_response = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          canned_response
        })
      )
    }
  }
]

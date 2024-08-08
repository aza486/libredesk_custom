import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'

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
    accessorKey: 'is_default',
    header: () => h('div', { class: 'text-center' }, 'Default'),
    cell: ({ row }) => {
      const isDefault = row.getValue('is_default')

      return h('div', { class: 'text-center' }, [
        h('input', {
          type: 'checkbox',
          checked: isDefault,
          disabled: true
        })
      ])
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    cell: ({ row }) => {
      const template = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          template
        })
      )
    }
  }
]

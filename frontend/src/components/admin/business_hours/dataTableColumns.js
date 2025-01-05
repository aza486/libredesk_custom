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
        accessorKey: 'created_at',
        header: function () {
            return h('div', { class: 'text-center' }, 'Created at')
        },
        cell: function ({ row }) {
            return h('div', { class: 'text-center font-medium' }, format(row.getValue('created_at'), 'PPpp'))
        }
    },
    {
        accessorKey: 'updated_at',
        header: function () {
            return h('div', { class: 'text-center' }, 'Updated at')
        },
        cell: function ({ row }) {
            return h('div', { class: 'text-center font-medium' }, format(row.getValue('updated_at'), 'PPpp'))
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

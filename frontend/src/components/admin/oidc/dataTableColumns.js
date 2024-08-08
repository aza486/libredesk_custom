import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'

export const columns = [

    {
        accessorKey: 'name',
        header: function () {
            return h('div', { class: 'text-center' }, 'Provider')
        },
        cell: function ({ row }) {
            return h('div', { class: 'text-center font-medium' }, row.getValue('name'))
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

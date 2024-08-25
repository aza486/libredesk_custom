import { h } from 'vue'
import TeamDataTableDropdown from '@/components/admin/team/teams/TeamDataTableDropdown.vue'
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
    accessorKey: 'updated_at',
    header: function () {
      return h('div', { class: 'text-center' }, 'Modified at')
    },
    cell: function ({ row }) {
      return h(
        'div',
        { class: 'text-center font-medium' },
        format(row.getValue('updated_at'), 'PPpp')
      )
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    cell: ({ row }) => {
      const team = row.original
      return h(
        'div',
        { class: 'relative' },
        h(TeamDataTableDropdown, {
          team
        })
      )
    }
  }
]

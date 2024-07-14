import { h } from 'vue'
import UserDataTableDropDown from '@/components/admin/team/UserDataTableDropDown.vue'

export const columns = [
  {
    accessorKey: 'first_name',
    header: function () {
      return h('div', { class: 'text-center' }, 'First name')
    },
    cell: function ({ row }) {
      const firstName = row.getValue('first_name')
      return h('div', { class: 'text-center font-medium' }, firstName)
    }
  },
  {
    accessorKey: 'last_name',
    header: function () {
      return h('div', { class: 'text-center' }, 'Last name')
    },
    cell: function ({ row }) {
      const lastName = row.getValue('last_name')
      return h('div', { class: 'text-center font-medium' }, lastName)
    }
  },
  {
    accessorKey: 'email',
    header: function () {
      return h('div', { class: 'text-center' }, 'Email')
    },
    cell: function ({ row }) {
      const email = row.getValue('email')
      return h('div', { class: 'text-center font-medium' }, email)
    }
  },
  {
    accessorKey: 'team_name',
    header: function () {
      return h('div', { class: 'text-center' }, 'Team name')
    },
    cell: function ({ row }) {
      const tName = row.getValue('team_name')
      return h('div', { class: 'text-center font-medium' }, tName)
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    cell: ({ row }) => {
      const user = row.original
      return h(
        'div',
        { class: 'relative' },
        h(UserDataTableDropDown, {
          user
        })
      )
    }
  }
]

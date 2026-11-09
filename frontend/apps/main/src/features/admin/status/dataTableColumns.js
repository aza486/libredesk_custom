import { h } from 'vue'
import dropdown from './dataTableDropdown.vue'
import { format } from 'date-fns'
import { CONVERSATION_DEFAULT_STATUSES_LIST } from '@/constants/conversation.js'

import { conversationStatusLabel } from '@/utils/conversation-status'

export const createColumns = (t, { onEdit } = {}) => [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.name'))
    },
    cell: function ({ row }) {
      const name = row.getValue('name')
      const isDefault = CONVERSATION_DEFAULT_STATUSES_LIST.includes(name)
      const label = conversationStatusLabel(name, t)
      return h('div', { class: 'text-center' },
        onEdit && !isDefault
          ? h('span', {
              class: 'text-foreground font-medium hover:underline cursor-pointer',
              onClick: () => onEdit(row.original)
            }, label)
          : label
      )
    }
  },
  {
    accessorKey: 'category',
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.category'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, t(`globals.terms.${row.getValue('category')}`))
    }
  },
  {
    accessorKey: 'created_at',
    enableGlobalFilter: false,
    header: function () {
      return h('div', { class: 'text-center' }, t('globals.terms.createdAt'))
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center' }, format(row.getValue('created_at'), 'PPpp'))
    }
  },
  {
    id: 'actions',
    enableHiding: false,
    enableSorting: false,
    cell: ({ row }) => {
      const status = row.original
      return h(
        'div',
        { class: 'relative' },
        h(dropdown, {
          status
        })
      )
    }
  }
]

import { h } from 'vue';

export const columns = [
  {
    accessorKey: 'name',
    header: function () {
      return h('div', { class: 'text-center' }, 'Name');
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('name'));
    },
  },
  {
    accessorKey: 'channel',
    header: function () {
      return h('div', { class: 'text-center' }, 'Channel');
    },
    cell: function ({ row }) {
      return h('div', { class: 'text-center font-medium' }, row.getValue('channel'));
    },
  },
];

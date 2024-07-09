import { h } from 'vue';

export const columns = [
  {
    accessorKey: 'first_name',
    header: function () {
      return h('div', { class: 'text-center' }, 'First name');
    },
    cell: function ({ row }) {
      const firstName = row.getValue('first_name')
      return h('div', { class: 'text-center font-medium' }, firstName);
    },
  },
  {
    accessorKey: 'last_name',
    header: function () {
      return h('div', { class: 'text-center' }, 'Last name');
    },
    cell: function ({ row }) {
      const lastName = row.getValue('last_name')
      return h('div', { class: 'text-center font-medium' }, lastName);
    },
  },
  {
    accessorKey: 'email',
    header: function () {
      return h('div', { class: 'text-center' }, 'Email');
    },
    cell: function ({ row }) {
      const email = row.getValue('email')
      return h('div', { class: 'text-center font-medium' }, email);
    },
  },
];

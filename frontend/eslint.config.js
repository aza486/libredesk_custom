import js from '@eslint/js'
import vue from 'eslint-plugin-vue'
import cypress from 'eslint-plugin-cypress'
import prettier from 'eslint-config-prettier'
import globals from 'globals'

// Preserve established wrapper/route component identities without exempting new files.
const establishedSingleWordComponents = [
  'apps/main/src/Root.vue',
  'apps/main/src/components/importer/Importer.vue',
  'apps/main/src/components/sidebar/Sidebar.vue',
  'apps/main/src/features/conversation/Conversation.vue',
  'apps/main/src/views/admin/agents/Agents.vue',
  'apps/main/src/views/admin/automations/Automation.vue',
  'apps/main/src/views/admin/general/General.vue',
  'apps/main/src/views/admin/macros/Macros.vue',
  'apps/main/src/views/admin/roles/Roles.vue',
  'apps/main/src/views/admin/teams/Teams.vue',
  'apps/main/src/views/admin/templates/Templates.vue',
  'apps/main/src/views/admin/webhooks/Webhooks.vue',
  'shared-ui/components/ui/accordion/Accordion.vue',
  'shared-ui/components/ui/alert/Alert.vue',
  'shared-ui/components/ui/avatar/Avatar.vue',
  'shared-ui/components/ui/badge/Badge.vue',
  'shared-ui/components/ui/breadcrumb/Breadcrumb.vue',
  'shared-ui/components/ui/breadcrumb/Custom.vue',
  'shared-ui/components/ui/button/Button.vue',
  'shared-ui/components/ui/calendar/Calendar.vue',
  'shared-ui/components/ui/card/Card.vue',
  'shared-ui/components/ui/checkbox/Checkbox.vue',
  'shared-ui/components/ui/collapsible/Collapsible.vue',
  'shared-ui/components/ui/command/Command.vue',
  'shared-ui/components/ui/dialog/Dialog.vue',
  'shared-ui/components/ui/error/Error.vue',
  'shared-ui/components/ui/input/Input.vue',
  'shared-ui/components/ui/label/Label.vue',
  'shared-ui/components/ui/popover/Popover.vue',
  'shared-ui/components/ui/select/Select.vue',
  'shared-ui/components/ui/separator/Separator.vue',
  'shared-ui/components/ui/sheet/Sheet.vue',
  'shared-ui/components/ui/sidebar/Sidebar.vue',
  'shared-ui/components/ui/skeleton/Skeleton.vue',
  'shared-ui/components/ui/sonner/Sonner.vue',
  'shared-ui/components/ui/spinner/Spinner.vue',
  'shared-ui/components/ui/stepper/Stepper.vue',
  'shared-ui/components/ui/switch/Switch.vue',
  'shared-ui/components/ui/table/Table.vue',
  'shared-ui/components/ui/tabs/Tabs.vue',
  'shared-ui/components/ui/textarea/Textarea.vue',
  'shared-ui/components/ui/toggle/Toggle.vue',
  'shared-ui/components/ui/tooltip/Tooltip.vue'
]

export default [
  {
    ignores: [
      '**/node_modules/**',
      '**/dist/**',
      '**/dist-ssr/**',
      '**/coverage/**',
      'cypress/videos/**',
      'cypress/screenshots/**'
    ]
  },
  js.configs.recommended,
  ...vue.configs['flat/essential'],
  {
    files: ['**/*.{js,jsx,cjs,mjs,vue}'],
    languageOptions: {
      globals: globals.browser,
      parserOptions: { ecmaVersion: 'latest' }
    },
    rules: {
      // Object rest intentionally omits these properties (form fixtures and forwarded props).
      'no-unused-vars': ['error', { ignoreRestSiblings: true }]
    }
  },
  {
    files: ['*.{js,cjs,mjs}', '**/*.test.js', 'cypress/**/*.{js,jsx}'],
    languageOptions: { globals: globals.node }
  },
  {
    files: [
      '**/__tests__/*.{cy,spec}.{js,ts,jsx,tsx}',
      'cypress/e2e/**/*.{cy,spec}.{js,ts,jsx,tsx}',
      'cypress/support/**/*.{js,ts,jsx,tsx}'
    ],
    ...cypress.configs.recommended,
    rules: {
      ...cypress.configs.recommended.rules,
      'vue/multi-word-component-names': 'off'
    }
  },
  {
    files: establishedSingleWordComponents,
    rules: {
      'vue/multi-word-component-names': [
        'error',
        {
          ignores: [
            ...new Set(
              establishedSingleWordComponents.map((file) =>
                file.split('/').at(-1).replace('.vue', '')
              )
            )
          ]
        }
      ]
    }
  },
  prettier
]

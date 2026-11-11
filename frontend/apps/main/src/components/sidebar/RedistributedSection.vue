<script setup>
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import { ChevronRight, ArrowLeftRight } from 'lucide-vue-next'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger
} from '@shared-ui/components/ui/collapsible'
import {
  SidebarMenuItem,
  SidebarMenuButton,
  SidebarMenuSub,
  SidebarMenuSubItem
} from '@shared-ui/components/ui/sidebar'
import SidebarCountBadge from './SidebarCountBadge.vue'
import SidebarFolders from './SidebarFolders.vue'

defineProps({ count: { type: Number, default: 0 } })
const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const open = useStorage('redistributedOpen', true)
function showAll() {
  router.push({ name: 'inbox', params: { type: 'customer-redistributed' }, query: {} })
}
</script>

<template>
  <Collapsible v-model:open="open" class="group/collapsible">
    <SidebarMenuItem>
      <CollapsibleTrigger as-child>
        <SidebarMenuButton
          :is-active="String(route.params.type || '').startsWith('customer-redistributed')"
        >
          <ArrowLeftRight />
          <span class="min-w-0 truncate">{{ t('conversation.routing.redistributed') }}</span>
          <span
            v-if="!open && count > 0"
            class="ml-2 size-2 shrink-0 rounded-full bg-success"
            aria-hidden="true"
          />
          <ChevronRight
            class="ml-auto transition-transform duration-200 group-data-[state=open]/collapsible:rotate-90"
          />
        </SidebarMenuButton>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <SidebarMenuSub>
          <SidebarMenuSubItem>
            <SidebarMenuButton
              size="sm"
              :is-active="route.params.type === 'customer-redistributed' && !route.query.folder"
              @click="showAll"
            >
              <span class="min-w-0 truncate">{{ t('conversation.routing.allRedistributed') }}</span>
              <SidebarCountBadge v-if="open" :count="count" />
            </SidebarMenuButton>
          </SidebarMenuSubItem>
          <SidebarFolders
            scope="redistributed"
            :scope-id="0"
            :scope-name="t('conversation.routing.redistributed')"
            :unfiled-label="t('conversation.routing.withoutFolder')"
            :add-label="t('conversation.routing.addFolder')"
          />
        </SidebarMenuSub>
      </CollapsibleContent>
    </SidebarMenuItem>
  </Collapsible>
</template>

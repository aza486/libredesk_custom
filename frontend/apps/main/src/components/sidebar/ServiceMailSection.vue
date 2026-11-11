<script setup>
import { computed, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useStorage } from '@vueuse/core'
import { ChevronRight, Wrench, Settings } from 'lucide-vue-next'
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
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription
} from '@shared-ui/components/ui/dialog'
import { useUserStore } from '@main/stores/user'
import SidebarCountBadge from './SidebarCountBadge.vue'
import SidebarFolders from './SidebarFolders.vue'
import ServiceEmailManager from './ServiceEmailManager.vue'

defineProps({ count: { type: Number, default: 0 } })
const { t } = useI18n()
const user = useUserStore()
const route = useRoute()
const router = useRouter()
const open = useStorage('serviceMailsOpen', true)
const manage = ref(false)
const canManage = computed(
  () =>
    user.roles.includes('Admin') ||
    user.can('service_emails:manage') ||
    user.can('general_settings:manage')
)
function showAll() {
  router.push({ name: 'inbox', params: { type: 'service-mails' }, query: {} })
}
</script>

<template>
  <Collapsible v-model:open="open" class="group/collapsible">
    <SidebarMenuItem>
      <CollapsibleTrigger as-child>
        <SidebarMenuButton :is-active="route.params.type === 'service-mails'">
          <Wrench /><span class="min-w-0 truncate">{{ t('serviceEmails.title') }}</span>
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
              :is-active="route.params.type === 'service-mails' && !route.query.folder"
              @click="showAll"
            >
              <span class="min-w-0 truncate">{{ t('serviceEmails.all') }}</span>
              <SidebarCountBadge v-if="open" :count="count" />
            </SidebarMenuButton>
          </SidebarMenuSubItem>
          <SidebarFolders
            scope="service_mail"
            :scope-id="0"
            :scope-name="t('serviceEmails.title')"
            :unfiled-label="t('serviceEmails.unfiled')"
            :add-label="t('serviceEmails.addFolder')"
          />
          <template v-if="canManage">
            <SidebarMenuSubItem
              ><SidebarMenuButton size="sm" class="text-muted-foreground" @click="manage = true"
                ><Settings class="size-3.5" /><span>{{
                  t('serviceEmails.manage')
                }}</span></SidebarMenuButton
              ></SidebarMenuSubItem
            >
          </template>
        </SidebarMenuSub>
      </CollapsibleContent>
    </SidebarMenuItem>
  </Collapsible>
  <Dialog v-model:open="manage">
    <DialogContent class="max-h-[90vh] overflow-y-auto">
      <DialogHeader
        ><DialogTitle>{{ t('serviceEmails.manage') }}</DialogTitle
        ><DialogDescription>{{ t('serviceEmails.title') }}</DialogDescription></DialogHeader
      >
      <ServiceEmailManager v-if="manage && canManage" @cancel="manage = false" />
    </DialogContent>
  </Dialog>
</template>

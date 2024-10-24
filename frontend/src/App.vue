<template>
  <Toaster />
  <TooltipProvider :delay-duration="200">
    <div class="font-inter">
      <div v-if="$route.path !== '/'">
        <div class="flex">
          <NavBar :is-collapsed="isCollapsed" :links="navLinks" :bottom-links="bottomLinks"
            class="shadow shadow-gray-300 h-screen" />
          <ResizablePanelGroup direction="horizontal" auto-save-id="app.vue.resizable.panel">
            <ResizableHandle id="resize-handle-1" />
            <ResizablePanel id="resize-panel-2">
              <div class="w-full h-screen">
                <RouterView />
              </div>
            </ResizablePanel>
          </ResizablePanelGroup>
        </div>
      </div>
      <div v-else>
        <RouterView />
      </div>
    </div>
  </TooltipProvider>
</template>

<script setup>
import { ref, onMounted, computed, onUnmounted } from 'vue'
import { RouterView, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useUserStore } from '@/stores/user'
import { initWS } from '@/websocket.js'
import { useEmitter } from '@/composables/useEmitter'
import { Toaster } from '@/components/ui/toast'
import { useToast } from '@/components/ui/toast/use-toast'
import { ResizableHandle, ResizablePanel, ResizablePanelGroup } from '@/components/ui/resizable'
import { TooltipProvider } from '@/components/ui/tooltip'
import { EMITTER_EVENTS } from '@/constants/emitterEvents.js'
import NavBar from '@/components/NavBar.vue'

const { t } = useI18n()
const { toast } = useToast()
const emitter = useEmitter()
const isCollapsed = ref(true)
const allNavLinks = [
  {
    title: t('navbar.dashboard'),
    to: '/dashboard',
    label: '',
    icon: 'lucide:layout-dashboard',
    permission: 'dashboard_global:read',
  },
  {
    title: t('navbar.conversations'),
    to: '/conversations',
    label: '',
    icon: 'lucide:message-circle-more'
  },
  {
    title: t('navbar.account'),
    to: '/account/profile',
    label: '',
    icon: 'lucide:circle-user-round'
  },
  {
    title: t('navbar.admin'),
    to: '/admin/general',
    label: '',
    icon: 'lucide:settings',
    permission: 'admin:read'
  }
]

const bottomLinks = [
  {
    to: '/logout',
    icon: 'lucide:log-out',
    title: 'Logout'
  }
]
const userStore = useUserStore()
const router = useRouter()

onMounted(() => {
  initToaster()
  getCurrentUser()
  initWS()
})

onUnmounted(() => {
  emitter.off(EMITTER_EVENTS.SHOW_TOAST, toast)
})

const getCurrentUser = () => {
  userStore.getCurrentUser().catch((err) => {
    if (err.response && err.response.status === 401) {
      router.push('/')
    }
  })
}

const initToaster = () => {
  emitter.on(EMITTER_EVENTS.SHOW_TOAST, toast)
}

const navLinks = computed(() =>
  allNavLinks.filter((link) =>
    !link.permission || (userStore.userPermissions.includes(link.permission) && link.permission)
  )
)
</script>

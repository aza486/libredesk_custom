<template>
  <Toaster />
  <TooltipProvider :delay-duration="200">
    <div class="bg-background text-foreground">
      <div v-if="$route.path !== '/'">
        <ResizablePanelGroup direction="horizontal" auto-save-id="app.vue.resizable.panel">
          <ResizablePanel id="resize-panel-1" collapsible :default-size="10" :collapsed-size="1" :min-size="7"
            :max-size="20" :class="cn(isCollapsed && 'min-w-[50px] transition-all duration-200 ease-in-out')"
            @expand="toggleNav(false)" @collapse="toggleNav(true)">
            <NavBar :is-collapsed="isCollapsed" :links="navLinks" />
          </ResizablePanel>
          <ResizableHandle id="resize-handle-1" />
          <ResizablePanel id="resize-panel-2">
            <div class="w-full h-screen">
              <RouterView />
            </div>
          </ResizablePanel>
        </ResizablePanelGroup>
      </div>
      <div v-else>
        <RouterView />
      </div>
    </div>
  </TooltipProvider>
</template>

<script setup>
import { ref, onMounted } from 'vue';
import { RouterView, useRouter } from 'vue-router';
import { cn } from '@/lib/utils';
import { useI18n } from 'vue-i18n';
import { useUserStore } from '@/stores/user';
import { initWS } from '@/websocket.js';

import { Toaster } from '@/components/ui/toast';
import NavBar from '@/components/NavBar.vue';
import {
  ResizableHandle,
  ResizablePanel,
  ResizablePanelGroup,
} from '@/components/ui/resizable';
import { TooltipProvider } from '@/components/ui/tooltip';

const { t } = useI18n();
const isCollapsed = ref(false);
const navLinks = ref([
  {
    title: t('navbar.dashboard'),
    component: 'dashboard',
    label: '',
    icon: 'lucide:layout-dashboard',
  },
  {
    title: t('navbar.conversations'),
    component: 'conversations',
    label: '',
    icon: 'lucide:message-circle-more',
  },
  {
    title: t('navbar.account'),
    component: 'account',
    label: '',
    icon: 'lucide:circle-user-round',
  },
  {
    title: t('navbar.admin'),
    component: 'admin',
    label: '',
    icon: 'lucide:settings',
  },
]);
const userStore = useUserStore();
const router = useRouter();

function toggleNav (v) {
  isCollapsed.value = v;
}

onMounted(() => {
  userStore.getCurrentUser().catch((err) => {
    if (err.response && err.response.status === 401) {
      router.push('/login');
    }
  });
  initWS();
});
</script>

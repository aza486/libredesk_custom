<template>
  <div class="flex h-screen">
    <template v-if="!isSearchRoute">
      <div class="w-[390px] border-r shrink-0 overflow-y-auto">
        <ConversationList />
      </div>
      <div class="grow overflow-hidden">
        <router-view v-slot="{ Component }">
          <keep-alive>
            <component :is="Component" />
          </keep-alive>
        </router-view>
      </div>
    </template>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import ConversationList from '@/features/conversation/list/ConversationList.vue'

const route = useRoute()
const isSearchRoute = computed(() => route.name === 'search')
</script>

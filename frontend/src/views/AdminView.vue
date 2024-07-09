<script setup>
import Admin from '@/components/admin/Admin.vue'
import Channel from '@/components/admin/Inbox.vue'
import Team from '@/components/admin/Team.vue'

const props = defineProps({
  page: {
    type: String,
    default: 'inboxes'
  }
});
</script>

<template>
    <Admin class="h-screen overflow-auto">
       <Channel v-if="props.page === 'inboxes'"></Channel>
       <Team v-if="props.page === 'team'"></Team>
    </Admin>
</template>

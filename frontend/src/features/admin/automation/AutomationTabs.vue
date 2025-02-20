<template>
  <Tabs default-value="new_conversation" v-model="selectedTab">
    <TabsList class="grid w-full grid-cols-3 mb-5">
      <TabsTrigger value="new_conversation">New conversation</TabsTrigger>
      <TabsTrigger value="conversation_update">Conversation update</TabsTrigger>
      <TabsTrigger value="time_trigger">Time triggers</TabsTrigger>
    </TabsList>
    <TabsContent value="new_conversation">
      <RuleTab type="new_conversation" helptext="Rules that run when a new conversation is created, drag and drop to reorder rules." />
    </TabsContent>
    <TabsContent value="conversation_update">
      <RuleTab type="conversation_update" helptext="Rules that run when a conversation is updated." />
    </TabsContent>
    <TabsContent value="time_trigger">
      <RuleTab type="time_trigger" helptext="Rules that run once an hour." />
    </TabsContent>
  </Tabs>
</template>

<script setup>
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import RuleTab from './RuleTab.vue'

const selectedTab = defineModel('automationsTab', {
  default: 'new_conversation',
  type: String,
  required: true
})
</script>
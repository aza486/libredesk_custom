<template>
  <Tabs default-value="conversation_creation">
    <TabsList class="grid w-full grid-cols-3 mb-5">
      <TabsTrigger value="conversation_creation"> New conversation </TabsTrigger>
      <TabsTrigger value="conversation_updates"> Conversation update </TabsTrigger>
      <TabsTrigger value="time_triggers"> Time triggers </TabsTrigger>
    </TabsList>
    <TabsContent value="conversation_creation">
      <RuleTab type="new_conversation" helptext="Rules that run when a new conversation is created."/>
    </TabsContent>
    <TabsContent value="conversation_updates">
      <RuleTab type="conversation_update" helptext="Rules that run when a conversation is updated."/>
    </TabsContent>
    <TabsContent value="time_triggers"> 
      <RuleTab type="time_trigger" helptext="Rules that run once an hour."/>
    </TabsContent>
  </Tabs>
</template>

<script setup>
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import RuleTab from './RuleTab.vue'
</script>

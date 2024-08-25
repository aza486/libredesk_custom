<template>
  <Tabs default-value="conversation_creation">
    <TabsList class="grid w-full grid-cols-3 mb-5">
      <TabsTrigger value="conversation_creation"> Conversation creation </TabsTrigger>
      <TabsTrigger value="conversation_updates"> Conversation updates </TabsTrigger>
      <TabsTrigger value="time_triggers"> Time triggers </TabsTrigger>
    </TabsList>
    <TabsContent value="conversation_creation">
      <TabConversationCreation />
    </TabsContent>
    <TabsContent value="conversation_updates">
      <TabConversationUpdate />
    </TabsContent>
    <TabsContent value="time_triggers">
      <TabTimeTrigger />
    </TabsContent>
  </Tabs>
</template>

<script setup>
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import TabConversationCreation from './TabConversationCreation.vue'
import TabConversationUpdate from './TabConversationUpdate.vue'
import TabTimeTrigger from './TabTimeTrigger.vue'
</script>

<script setup lang="ts">
import { TriangleAlert } from 'lucide-vue-next';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

defineProps({
    errorMessage: String,
});
</script>

<template>
  <Alert variant="destructive" v-if="errorMessage">
    <TriangleAlert class="w-4 h-4" />
    <AlertTitle>Error</AlertTitle>
    <AlertDescription>
      {{ errorMessage }}
    </AlertDescription>
  </Alert>
</template>
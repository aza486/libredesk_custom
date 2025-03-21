<template>
  <div class="overflow-y-auto h-screen">
    <div class="p-6 sm:p-8 min-h-full flex flex-col">
      <div
        class="flex flex-col items-center justify-center flex-grow"
        v-if="$route.path === '/admin'"
      >
        <div>Select a section from the sidebar</div>
      </div>
      <router-view class="flex-grow" />
    </div>
  </div>
</template>

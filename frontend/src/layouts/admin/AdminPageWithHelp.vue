<template>
  <div class="flex justify-between">
    <div class="w-8/12">
      <slot name="content" />
    </div>
    <div class="rounded-lg w-3/12 p-2 space-y-2 h-max-content">
      <slot name="help" />
    </div>
  </div>
</template>

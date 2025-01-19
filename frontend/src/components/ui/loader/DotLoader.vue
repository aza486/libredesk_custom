<template>
  <div class="flex flex-col items-center justify-center py-8 text-gray-600 dark:text-gray-300">
    <div class="dot-spinner mb-4">
      <div class="dot"></div>
      <div class="dot"></div>
      <div class="dot"></div>
      <div class="dot"></div>
    </div>
    <p class="text-sm font-medium">Loading...</p>
  </div>
</template>

<style scoped>
.dot-spinner {
  display: flex;
  justify-content: center;
  align-items: center;
}

.dot {
  width: 10px;
  height: 10px;
  margin: 0 5px;
  background-color: currentColor;
  border-radius: 50%;
  display: inline-block;
  animation: dot-flashing 1s infinite alternate;
}

.dot:nth-child(2) {
  animation-delay: 0.2s;
}

.dot:nth-child(3) {
  animation-delay: 0.4s;
}

.dot:nth-child(4) {
  animation-delay: 0.6s;
}

@keyframes dot-flashing {
  0% {
    opacity: 0.2;
  }
  100% {
    opacity: 1;
  }
}
</style>

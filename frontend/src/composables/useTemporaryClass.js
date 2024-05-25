// Applies a temporary class to an element.
import { ref, onUnmounted } from 'vue';

export function useTemporaryClass (containerID, className, timeMs = 500) {
    const container = ref(document.getElementById(containerID));
    container.value.classList.add(className);
    setTimeout(() => {
        container.value.classList.remove(className);
    }, timeMs);

    onUnmounted(() => {
        container.value = null;
    });
}

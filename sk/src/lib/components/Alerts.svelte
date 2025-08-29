<script lang="ts" module>
  import { writable } from 'svelte/store';
  
  interface Alert {
    id: string;
    type: 'success' | 'error' | 'info' | 'warning';
    message: string;
  }
  
  const alertStore = writable<Alert[]>([]);
  
  function addAlert(type: Alert['type'], message: string) {
    const id = Math.random().toString(36).substring(7);
    alertStore.update(alerts => [...alerts, { id, type, message }]);
    
    // Auto-remove after 5 seconds
    setTimeout(() => {
      alertStore.update(alerts => alerts.filter(a => a.id !== id));
    }, 5000);
  }
  
  export const alerts = {
    success: (message: string) => addAlert('success', message),
    error: (message: string) => addAlert('error', message),
    info: (message: string) => addAlert('info', message),
    warning: (message: string) => addAlert('warning', message),
  };
</script>

<script lang="ts">
  import { fly } from 'svelte/transition';
  
  let alertList = $state<Alert[]>([]);
  
  $effect(() => {
    alertStore.subscribe(value => {
      alertList = value;
    });
  });
</script>

{#if alertList.length > 0}
  <div class="fixed top-4 right-4 z-50 space-y-2">
    {#each alertList as alert (alert.id)}
      <div
        transition:fly={{ x: 100, duration: 300 }}
        class="px-4 py-3 rounded-md shadow-md max-w-sm {alert.type === 'success' ? 'bg-green-100 text-green-900' : ''} {alert.type === 'error' ? 'bg-red-100 text-red-900' : ''} {alert.type === 'info' ? 'bg-blue-100 text-blue-900' : ''} {alert.type === 'warning' ? 'bg-yellow-100 text-yellow-900' : ''}"
      >
        {alert.message}
      </div>
    {/each}
  </div>
{/if}
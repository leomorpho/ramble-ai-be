<script lang="ts">
  import { Button } from '$lib/components/ui/button/index.js';
  import { Input } from '$lib/components/ui/input/index.js';
  import { Key, Copy, Plus, AlertTriangle, CheckCircle, Shield, Info } from 'lucide-svelte';
  import * as AlertDialog from '$lib/components/ui/alert-dialog/index.js';
  import * as Dialog from '$lib/components/ui/dialog/index.js';
  import { pb } from '$lib/pocketbase.js';
  import { authStore } from '$lib/stores/authClient.svelte.js';
  import { onMount } from 'svelte';

  // Component state
  let apiKeys = $state([]);
  let isLoading = $state(true);
  let isGenerating = $state(false);
  let error = $state<string | null>(null);
  let success = $state<string | null>(null);
  
  // New API key display state
  let newGeneratedKey = $state<string | null>(null);
  let keyVisibilityCountdown = $state<number>(0);
  
  // Copy feedback state
  let copiedKeyId = $state<string | null>(null);
  
  // Confirmation dialog state
  let showConfirmDialog = $state(false);
  
  // Help dialog state
  let showHelpDialog = $state(false);

  // Load API keys on mount
  onMount(async () => {
    await loadAPIKeys();
  });

  // Load user's API keys
  async function loadAPIKeys() {
    if (!authStore.user?.id) {
      console.log('No user ID available');
      return;
    }
    
    console.log('Loading API keys for user:', authStore.user.id);
    console.log('Auth store state:', { isValid: pb.authStore.isValid, token: pb.authStore.token ? 'present' : 'missing' });
    
    try {
      isLoading = true;
      error = null;
      
      // Try to list all collections first to verify access
      console.log('Testing PocketBase connection...');
      
      // Simple query without any parameters to see if collection is accessible
      console.log('Attempting to query api_keys collection...');
      const records = await pb.collection('api_keys').getList(1, 10);
      
      console.log('API keys query successful:', records);
      apiKeys = records.items || [];
    } catch (err: any) {
      console.error('Failed to load API keys:', err);
      console.error('Error response:', err.response);
      console.error('Error data:', err.data);
      console.error('Full error object:', JSON.stringify(err, null, 2));
      
      // More specific error handling based on what we see
      if (err.status === 400) {
        error = `Bad Request (400): ${err.data?.message || err.message || 'Invalid query parameters'}`;
      } else if (err.status === 403) {
        error = 'Access denied. Check if you have permission to access API keys.';
      } else if (err.status === 404) {
        error = 'API keys collection not found.';
      } else {
        error = `Error ${err.status}: ${err.message || 'Unknown error'}`;
      }
      
      // Set empty array so component doesn't break
      apiKeys = [];
    } finally {
      isLoading = false;
    }
  }

  // Show confirmation dialog or generate directly
  function handleGenerateClick() {
    const hasExistingKey = apiKeys.length > 0;
    
    if (hasExistingKey) {
      showConfirmDialog = true;
    } else {
      generateAPIKey();
    }
  }

  // Generate new API key (actual generation)
  async function generateAPIKey() {
    const hasExistingKey = apiKeys.length > 0;
    showConfirmDialog = false;

    try {
      isGenerating = true;
      error = null;
      success = null;

      // Delete all existing keys (simpler than deactivating)
      for (const key of apiKeys) {
        try {
          await pb.collection('api_keys').delete(key.id);
        } catch (deleteErr) {
          console.warn('Failed to delete existing key:', deleteErr);
        }
      }

      // Call the custom generate endpoint - try the correct path
      const response = await pb.send('/api/generate-api-key', {
        method: 'POST'
      });

      if (response.api_key) {
        newGeneratedKey = response.api_key;
        keyVisibilityCountdown = 60; // Start 60 second countdown
        success = hasExistingKey 
          ? 'New API key generated! Your old key has been replaced. Copy it from below - you won\'t be able to see it again.'
          : 'API key generated successfully! Copy it from below - you won\'t be able to see it again.';
        
        // Reload keys list to include the new one
        await loadAPIKeys();
      }
    } catch (err: any) {
      console.error('Failed to generate API key:', err);
      error = err.message || 'Failed to generate API key';
    } finally {
      isGenerating = false;
    }
  }

  // Copy API key to clipboard
  async function copyToClipboard(key: string, keyId: string) {
    try {
      await navigator.clipboard.writeText(key);
      copiedKeyId = keyId;
      
      // Clear copy feedback after 2 seconds
      setTimeout(() => {
        copiedKeyId = null;
      }, 2000);
    } catch (err) {
      console.error('Failed to copy to clipboard:', err);
      error = 'Failed to copy to clipboard';
    }
  }


  // Format date for display
  function formatDate(dateString: string): string {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    });
  }

  // Mask API key for display (show first 8 chars + ...)
  function maskKey(keyName: string): string {
    return 'ra-••••••••';
  }

  // Determine if we should show the actual key or masked version
  function getDisplayKey(apiKey: any): string {
    // If we have a newly generated key and this is the most recent API key, show the full key
    if (newGeneratedKey && apiKeys.length > 0 && apiKey.id === apiKeys[0].id) {
      return newGeneratedKey;
    }
    return maskKey(apiKey.key_hash);
  }

  // Check if this API key is the newly generated one
  function isNewlyGenerated(apiKey: any): boolean {
    return newGeneratedKey !== null && apiKeys.length > 0 && apiKey.id === apiKeys[0].id;
  }

  // Format countdown time as MM:SS
  function formatCountdown(seconds: number): string {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${mins}:${secs.toString().padStart(2, '0')}`;
  }

  // Clear success/error messages after timeout
  $effect(() => {
    if (success || error) {
      const timeout = setTimeout(() => {
        success = null;
        error = null;
      }, 5000);
      return () => clearTimeout(timeout);
    }
  });

  // Countdown timer for key visibility
  $effect(() => {
    if (keyVisibilityCountdown > 0) {
      const timer = setInterval(() => {
        keyVisibilityCountdown--;
        if (keyVisibilityCountdown <= 0) {
          newGeneratedKey = null;
        }
      }, 1000);
      return () => clearInterval(timer);
    }
  });
</script>

<div class="border rounded-lg p-6">
  <div class="flex items-center justify-between mb-4">
    <div class="flex items-center gap-2">
      <h3 class="text-lg font-semibold">API Key Management</h3>
      <Button
        variant="ghost"
        size="sm"
        onclick={() => showHelpDialog = true}
        class="w-6 h-6 p-0 text-muted-foreground hover:text-foreground"
      >
        <Info class="w-4 h-4" />
      </Button>
    </div>
    <Button
      size="sm"
      onclick={handleGenerateClick}
      disabled={isGenerating}
      class="flex items-center gap-2"
    >
      <Plus class="w-4 h-4" />
      {apiKeys.length > 0 ? 'Replace Key' : 'Generate Key'}
    </Button>
  </div>

  <!-- Success/Error Messages -->
  {#if success}
    <div class="mb-4 p-3 bg-green-50 dark:bg-green-950/30 border border-green-200 dark:border-green-800 rounded-lg flex items-start gap-2">
      <CheckCircle class="w-4 h-4 text-green-600 dark:text-green-400 flex-shrink-0 mt-0.5" />
      <p class="text-sm text-green-700 dark:text-green-300">{success}</p>
    </div>
  {/if}

  {#if error}
    <div class="mb-4 p-3 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg flex items-start gap-2">
      <AlertTriangle class="w-4 h-4 text-red-600 dark:text-red-400 flex-shrink-0 mt-0.5" />
      <p class="text-sm text-red-700 dark:text-red-300">{error}</p>
    </div>
  {/if}



  <!-- API Keys List -->
  {#if isLoading}
    <div class="text-center py-8">
      <div class="animate-spin rounded-full h-8 w-8 border-2 border-primary border-t-transparent mx-auto mb-2"></div>
      <p class="text-sm text-muted-foreground">Loading API keys...</p>
    </div>
  {:else if apiKeys.length === 0}
    <div class="text-center py-8">
      <Key class="w-12 h-12 text-muted-foreground mx-auto mb-3 opacity-50" />
      <h4 class="text-sm font-medium text-foreground mb-1">No API Key Yet</h4>
      <p class="text-xs text-muted-foreground mb-4">Generate your API key to start using the Ramble AI desktop application</p>
      <Button
        size="sm"
        onclick={handleGenerateClick}
        class="flex items-center gap-2"
      >
        <Plus class="w-4 h-4" />
        Generate API Key
      </Button>
    </div>
  {:else}
    <div class="space-y-4">
      {#each apiKeys as apiKey}
        {@const isNewKey = isNewlyGenerated(apiKey)}
        {@const displayKey = getDisplayKey(apiKey)}
        <div class="p-4 rounded-lg border {isNewKey ? 'bg-yellow-50 dark:bg-yellow-950/30 border-yellow-200 dark:border-yellow-800' : 'bg-muted/30'}">
          <div class="flex items-center justify-between">
            <div class="flex-1">
              <div class="flex items-center gap-2">
                <code class="text-sm font-mono">{displayKey}</code>
                {#if isNewKey}
                  <Button
                    size="sm"
                    onclick={() => copyToClipboard(displayKey, apiKey.id)}
                    class="flex items-center gap-1 h-6 px-2 text-xs"
                  >
                    <Copy class="w-3 h-3" />
                    {copiedKeyId === apiKey.id ? 'Copied!' : 'Copy'}
                  </Button>
                  {#if keyVisibilityCountdown > 0}
                    <span class="text-xs text-yellow-700 dark:text-yellow-300 bg-yellow-100 dark:bg-yellow-900/50 px-2 py-1 rounded">
                      Visible for {formatCountdown(keyVisibilityCountdown)}
                    </span>
                  {/if}
                {/if}
              </div>
              <div class="flex items-center gap-3 mt-2">
                <span class="px-2 py-0.5 bg-green-50 dark:bg-green-950/30 text-green-800 dark:text-green-200 rounded text-xs font-medium">
                  Active
                </span>
                <span class="text-xs text-muted-foreground">
                  Created {formatDate(apiKey.created)}
                </span>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  {/if}

</div>

<!-- Confirmation Dialog -->
<AlertDialog.Root bind:open={showConfirmDialog}>
  <AlertDialog.Content class="sm:max-w-[425px]">
    <AlertDialog.Header>
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-full bg-amber-50 dark:bg-amber-950/30 flex items-center justify-center">
          <Shield class="w-5 h-5 text-amber-600 dark:text-amber-400" />
        </div>
        <div>
          <AlertDialog.Title class="text-left">Reveal live API key</AlertDialog.Title>
        </div>
      </div>
    </AlertDialog.Header>
    <div class="py-4">
      <AlertDialog.Description class="text-left leading-relaxed">
        This key can only be revealed once to keep your account secure. If you lose it, you must rotate the key or create another one.
        <br><br>
        <strong>Your current API key will stop working immediately</strong> and cannot be recovered.
        <br><br>
        Are you sure you want to reveal your new key?
      </AlertDialog.Description>
    </div>
    <AlertDialog.Footer class="flex-row justify-end gap-2">
      <AlertDialog.Cancel disabled={isGenerating}>
        Cancel
      </AlertDialog.Cancel>
      <AlertDialog.Action 
        onclick={generateAPIKey}
        disabled={isGenerating}
        class="bg-destructive text-destructive-foreground hover:bg-destructive/90"
      >
        {isGenerating ? 'Generating...' : 'Reveal Key'}
      </AlertDialog.Action>
    </AlertDialog.Footer>
  </AlertDialog.Content>
</AlertDialog.Root>

<!-- Help Dialog -->
<Dialog.Root bind:open={showHelpDialog}>
  <Dialog.Content class="sm:max-w-[400px]">
    <Dialog.Header>
      <Dialog.Title class="flex items-center gap-2">
        <Info class="w-5 h-5 text-blue-600 dark:text-blue-400" />
        Using Your API Key
      </Dialog.Title>
    </Dialog.Header>
    <div class="space-y-3 py-4">
      <div class="text-sm text-muted-foreground space-y-2">
        <p>• Copy your key into Ramble AI desktop app settings</p>
        <p>• Only one active key at a time - new keys replace old ones</p>
        <p>• Keys are shown once only - copy immediately</p>
        <p>• Keep secure - anyone with access can use your account</p>
      </div>
    </div>
  </Dialog.Content>
</Dialog.Root>
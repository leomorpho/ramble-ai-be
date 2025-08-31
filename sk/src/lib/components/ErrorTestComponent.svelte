<script lang="ts">
	import { Logger } from '$lib/logger';

	let errorMessage = $state('');

	function triggerConsoleError() {
		console.error('This is a test console error', { timestamp: new Date().toISOString() });
		Logger.error('Test error logged through Logger utility', new Error('Test error'));
		errorMessage = 'Console error triggered - check your Sentry dashboard';
	}

	function triggerException() {
		try {
			// This will throw an error
			throw new Error('This is a test exception for Sentry');
		} catch (error) {
			Logger.captureException(error as Error, { 
				component: 'ErrorTestComponent',
				action: 'manual_trigger'
			});
			errorMessage = 'Exception captured - check your Sentry dashboard';
		}
	}

	function triggerWarning() {
		console.warn('This is a test warning', { component: 'ErrorTestComponent' });
		Logger.warn('Test warning logged through Logger utility', { component: 'ErrorTestComponent' });
		errorMessage = 'Warning logged - check your Sentry dashboard';
	}

	function setUserContext() {
		Logger.setUser({
			id: 'test-user-123',
			email: 'test@example.com',
			username: 'testuser'
		});
		
		Logger.setTag('test_component', 'true');
		Logger.setContext('test_info', {
			browser: navigator.userAgent,
			url: window.location.href,
			timestamp: new Date().toISOString()
		});
		
		errorMessage = 'User context set for error tracking';
	}

	function clearMessage() {
		errorMessage = '';
	}
</script>

<div class="p-4 border border-gray-300 rounded-lg max-w-md mx-auto">
	<h3 class="text-lg font-bold mb-4">Sentry Error Testing</h3>
	
	<div class="space-y-2">
		<button 
			onclick={triggerConsoleError}
			class="w-full px-4 py-2 bg-red-500 text-white rounded hover:bg-red-600"
		>
			Test Console Error
		</button>
		
		<button 
			onclick={triggerException}
			class="w-full px-4 py-2 bg-orange-500 text-white rounded hover:bg-orange-600"
		>
			Test Exception Capture
		</button>
		
		<button 
			onclick={triggerWarning}
			class="w-full px-4 py-2 bg-yellow-500 text-white rounded hover:bg-yellow-600"
		>
			Test Warning Log
		</button>
		
		<button 
			onclick={setUserContext}
			class="w-full px-4 py-2 bg-blue-500 text-white rounded hover:bg-blue-600"
		>
			Set User Context
		</button>
		
		{#if errorMessage}
			<div class="mt-4 p-2 bg-green-100 text-green-800 rounded">
				<p>{errorMessage}</p>
				<button 
					onclick={clearMessage}
					class="mt-2 px-2 py-1 bg-green-500 text-white text-sm rounded hover:bg-green-600"
				>
					Clear
				</button>
			</div>
		{/if}
	</div>
	
	<div class="mt-4 text-sm text-gray-600">
		<p><strong>Instructions:</strong></p>
		<ol class="list-decimal list-inside mt-1">
			<li>Set up your Sentry DSN in .env file</li>
			<li>Click the test buttons above</li>
			<li>Check your Sentry dashboard for captured errors</li>
		</ol>
	</div>
</div>
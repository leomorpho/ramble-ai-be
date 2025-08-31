# Sentry Error Monitoring Setup

This guide walks you through setting up Sentry error monitoring for your SvelteKit static application.

## 1. Create Sentry Account

1. Go to [sentry.io](https://sentry.io) and create an account
2. Create a new project:
   - Choose **SvelteKit** as the platform
   - Give your project a name
   - Choose your team/organization

## 2. Configure Environment Variables

1. Copy `.env.example` to `.env`:
   ```bash
   cp .env.example .env
   ```

2. Update your `.env` file with your Sentry DSN:
   ```bash
   # Get this from your Sentry project settings
   VITE_SENTRY_DSN=https://your-actual-dsn@sentry.io/project-id
   VITE_ENVIRONMENT=development
   ```

3. Find your DSN in Sentry:
   - Go to **Settings** → **Projects** → **[Your Project]**
   - Click **Client Keys (DSN)**
   - Copy the **DSN** value

## 3. Features Included

### ✅ Already Configured

- **Console Error Capturing**: Automatically captures console.error, console.warn, etc.
- **Exception Handling**: Catches unhandled exceptions and promises
- **Structured Logging**: Logger utility for different log levels
- **Session Replay**: Records user sessions when errors occur (10% sample rate)
- **Performance Monitoring**: Tracks performance metrics (10% sample rate)
- **User Context**: Ability to associate errors with specific users
- **Environment Detection**: Automatically detects dev/prod environments

### 📋 Error Capturing Includes

- **Unhandled JavaScript exceptions**
- **Console errors and warnings**
- **Failed network requests**
- **Performance issues**
- **User interactions leading to errors**

## 4. Usage Examples

### Basic Error Logging

```typescript
import { Logger } from '$lib/logger';

// Log an error with context
Logger.error('Payment failed', error, { 
  userId: user.id,
  paymentAmount: amount 
});

// Log a warning
Logger.warn('API rate limit approaching', { 
  remaining: rateLimitRemaining 
});

// Log info for debugging
Logger.info('User action completed', { 
  action: 'subscription_created',
  userId: user.id 
});
```

### Set User Context

```typescript
import { Logger } from '$lib/logger';

// Set user information for all subsequent errors
Logger.setUser({
  id: user.id,
  email: user.email,
  username: user.username
});

// Add custom tags
Logger.setTag('subscription_tier', 'premium');

// Add extra context
Logger.setContext('payment_info', {
  lastPayment: lastPaymentDate,
  subscription: subscriptionType
});
```

### Manual Error Capture

```typescript
import { Logger } from '$lib/logger';

try {
  await riskyOperation();
} catch (error) {
  Logger.captureException(error, {
    component: 'PaymentProcessor',
    operation: 'charge_card'
  });
}
```

## 5. Testing Your Setup

1. Start your development server:
   ```bash
   npm run dev
   ```

2. Add the test component to any page:
   ```svelte
   <script>
     import ErrorTestComponent from '$lib/components/ErrorTestComponent.svelte';
   </script>
   
   <ErrorTestComponent />
   ```

3. Click the test buttons and check your Sentry dashboard for captured errors

## 6. Production Setup

### Environment Variables for Production

```bash
VITE_SENTRY_DSN=https://your-production-dsn@sentry.io/project-id
VITE_ENVIRONMENT=production
```

### Optional: Source Maps Upload

1. Get a Sentry auth token:
   - Go to **Settings** → **Auth Tokens**
   - Create a new token with `project:write` permissions

2. Add to your environment:
   ```bash
   SENTRY_AUTH_TOKEN=your-auth-token
   ```

3. Update `vite.config.ts` to enable source maps upload:
   ```typescript
   svelteKitVitePlugin({
     org: 'your-org-slug',
     project: 'your-project-slug',
     sourceMapsUploadOptions: {
       enabled: true
     }
   })
   ```

## 7. Monitoring Dashboard

In your Sentry dashboard, you can:

- **View Errors**: See all captured errors with stack traces
- **Session Replays**: Watch video-like replays of user sessions with errors
- **Performance**: Monitor page load times and API response times
- **Alerts**: Set up notifications for new errors or performance issues
- **Releases**: Track errors across different deployments

## 8. Best Practices

### Error Boundaries
Create error boundaries for critical components:

```svelte
<!-- ErrorBoundary.svelte -->
<script lang="ts">
  import { Logger } from '$lib/logger';
  
  let { children } = $props();
  let hasError = $state(false);
  let error = $state<Error | null>(null);

  function handleError(event: ErrorEvent) {
    hasError = true;
    error = event.error;
    Logger.captureException(event.error, { component: 'ErrorBoundary' });
  }
</script>

<svelte:window onerror={handleError} />

{#if hasError}
  <div class="error-fallback">
    <h2>Something went wrong</h2>
    <p>We've been notified about this error.</p>
  </div>
{:else}
  {@render children()}
{/if}
```

### Context Setting
Set context early in your app lifecycle:

```typescript
// In your main layout or app initialization
import { Logger } from '$lib/logger';

// Set user context when user logs in
function setUserContext(user) {
  Logger.setUser({
    id: user.id,
    email: user.email
  });
  
  Logger.setContext('app_info', {
    version: '1.0.0',
    buildDate: import.meta.env.VITE_BUILD_DATE
  });
}
```

### Filtering Sensitive Data

The current setup automatically filters sensitive information, but you can customize:

```typescript
// In hooks.client.ts
beforeSend(event) {
  // Remove sensitive data
  if (event.request?.headers) {
    delete event.request.headers.Authorization;
  }
  
  // Filter out specific error types in development
  if (dev && event.exception?.values?.[0]?.type === 'ChunkLoadError') {
    return null; // Don't send to Sentry
  }
  
  return event;
}
```

## 9. Troubleshooting

### Common Issues

1. **"Sentry DSN not configured"** in console
   - Check that your `.env` file exists and has the correct DSN
   - Restart your dev server after updating environment variables

2. **Errors not appearing in Sentry**
   - Verify your DSN is correct
   - Check your project's rate limits in Sentry settings
   - Make sure the error is actually being thrown (check browser console)

3. **Too many errors being captured**
   - Adjust sample rates in `hooks.client.ts`
   - Add filtering in the `beforeSend` function

### Debug Mode

Enable debug mode during development:

```typescript
// In hooks.client.ts
Sentry.init({
  debug: true, // Enable debug logging
  // ... other options
});
```

This will log Sentry's internal operations to the console.
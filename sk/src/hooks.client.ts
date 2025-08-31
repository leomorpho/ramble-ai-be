import { handleErrorWithSentry, replayIntegration, consoleLoggingIntegration } from '@sentry/sveltekit';
import * as Sentry from '@sentry/sveltekit';
import { dev } from '$app/environment';

const sentryDsn = import.meta.env.VITE_SENTRY_DSN;
const environment = import.meta.env.VITE_ENVIRONMENT || (dev ? 'development' : 'production');

// Only initialize Sentry if DSN is provided
if (sentryDsn && sentryDsn !== 'https://your-dsn@sentry.io/project-id') {
	Sentry.init({
		dsn: sentryDsn,
		
		// Capture 10% of sessions for performance monitoring
		tracesSampleRate: 0.1,

		// Capture 10% of sessions for session replay
		replaysSessionSampleRate: 0.1,

		// Capture 100% of sessions when an error occurs
		replaysOnErrorSampleRate: 1.0,

		// Set to false in production
		debug: dev,

		environment,

	integrations: [
		// Enable Session Replay (optional)
		replayIntegration({
			maskAllText: false,
			blockAllMedia: false,
		}),
		
		// Enable console logging capture
		consoleLoggingIntegration({
			levels: ['info', 'warn', 'error', 'debug', 'log']
		}),
	],

	// Enable logs (optional)
	enableLogs: true,

		beforeSend(event) {
			// Filter out development errors or add custom logic
			if (event.environment === 'development') {
				console.log('Sentry event:', event);
			}
			return event;
		}
	});
} else {
	console.log('Sentry DSN not configured - error monitoring disabled');
}

// Custom error handler
export const handleError = handleErrorWithSentry();
import * as Sentry from '@sentry/sveltekit';

/**
 * Logger utility for Sentry integration
 * Provides convenient methods to log different levels of messages
 */
export class Logger {
	/**
	 * Log an info message
	 */
	static info(message: string, extra?: Record<string, any>) {
		console.info(message, extra);
		Sentry.logger.info(message, extra);
	}

	/**
	 * Log a warning message
	 */
	static warn(message: string, extra?: Record<string, any>) {
		console.warn(message, extra);
		Sentry.logger.warn(message, extra);
	}

	/**
	 * Log an error message
	 */
	static error(message: string, error?: Error, extra?: Record<string, any>) {
		console.error(message, error, extra);
		Sentry.logger.error(message, { error, ...extra });
		
		// Also capture as exception if an Error object is provided
		if (error) {
			Sentry.captureException(error, {
				tags: { component: 'logger' },
				extra
			});
		}
	}

	/**
	 * Log a debug message
	 */
	static debug(message: string, extra?: Record<string, any>) {
		console.debug(message, extra);
		Sentry.logger.debug(message, extra);
	}

	/**
	 * Log a trace message
	 */
	static trace(message: string, extra?: Record<string, any>) {
		console.trace(message, extra);
		Sentry.logger.trace(message, extra);
	}

	/**
	 * Set user context for error tracking
	 */
	static setUser(user: { id: string; email?: string; username?: string }) {
		Sentry.setUser(user);
	}

	/**
	 * Set custom tag for error tracking
	 */
	static setTag(key: string, value: string) {
		Sentry.setTag(key, value);
	}

	/**
	 * Set extra context for error tracking
	 */
	static setContext(key: string, context: Record<string, any>) {
		Sentry.setContext(key, context);
	}

	/**
	 * Manually capture an exception
	 */
	static captureException(error: Error, context?: Record<string, any>) {
		Sentry.captureException(error, context);
	}

	/**
	 * Manually capture a message
	 */
	static captureMessage(message: string, level: 'info' | 'warning' | 'error' = 'info') {
		Sentry.captureMessage(message, level);
	}
}
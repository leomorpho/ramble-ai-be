// Static adapter doesn't use server-side hooks, so we disable Sentry server integration
// This prevents proxy errors during development

// Simple error handler for static builds - no Sentry server integration needed
export const handleError = ({ error }: { error: any }) => {
	// In development, log the error
	if (process.env.NODE_ENV === 'development') {
		console.error('SvelteKit error:', error);
	}
	
	// Return a generic message for production
	return {
		message: 'An unexpected error occurred'
	};
};
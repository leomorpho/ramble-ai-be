// Application configuration
export const config = {
	app: {
		name: 'Ramble',
		description: 'AI Script Optimization for Talking Head Videos'
	},
	
	// Download URL for the app
	download: {
		url: 'https://github.com/your-username/ramble-ai/releases/latest'
	},
	
	// Get current year dynamically
	getCurrentYear: () => new Date().getFullYear()
} as const;
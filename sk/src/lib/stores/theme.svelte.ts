import { browser } from '$app/environment';

type Theme = 'light' | 'dark';

class ThemeStore {
	private _theme = $state<Theme>('light');

	constructor() {
		if (browser) {
			// Get theme from localStorage or system preference
			const stored = localStorage.getItem('theme') as Theme;
			const systemPreference = window.matchMedia('(prefers-color-scheme: dark)').matches
				? 'dark'
				: 'light';

			this._theme = stored || systemPreference;
			this.updateDOM();

			// Listen for system theme changes
			window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
				if (!localStorage.getItem('theme')) {
					this._theme = e.matches ? 'dark' : 'light';
					this.updateDOM();
				}
			});
		}
	}

	get theme(): Theme {
		return this._theme;
	}

	set theme(value: Theme) {
		this._theme = value;
		if (browser) {
			localStorage.setItem('theme', value);
			this.updateDOM();
		}
	}

	toggle() {
		this.theme = this._theme === 'light' ? 'dark' : 'light';
	}

	private updateDOM() {
		const root = document.documentElement;
		root.classList.remove('light', 'dark');
		root.classList.add(this._theme);
	}
}

export const themeStore = new ThemeStore();

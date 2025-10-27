import { browser } from '$app/environment';

type Theme = 'light' | 'dark' | 'system';

class ThemeStore {
	current = $state<Theme>('system');

	constructor() {
		if (browser) {
			// Load saved theme from localStorage
			const saved = localStorage.getItem('theme') as Theme;
			if (saved && ['light', 'dark', 'system'].includes(saved)) {
				this.current = saved;
			}
			// Apply theme immediately
			this.applyTheme();

			// Listen for system theme changes
			window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
				if (this.current === 'system') {
					this.applyTheme();
				}
			});
		}
	}

	setTheme(theme: Theme) {
		this.current = theme;
		if (browser) {
			localStorage.setItem('theme', theme);
			this.applyTheme();
		}
	}

	applyTheme() {
		if (!browser) return;

		const root = document.documentElement;

		// Remove both attribute and class first
		root.removeAttribute('data-theme');
		root.classList.remove('dark', 'light');

		if (this.current === 'light') {
			root.setAttribute('data-theme', 'light');
			root.classList.add('light');
		} else if (this.current === 'dark') {
			root.setAttribute('data-theme', 'dark');
			root.classList.add('dark');
		} else {
			// System mode - check actual preference
			const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
			if (isDark) {
				root.classList.add('dark');
			}
		}
	}

	toggle() {
		if (this.current === 'light') {
			this.setTheme('dark');
		} else if (this.current === 'dark') {
			this.setTheme('system');
		} else {
			this.setTheme('light');
		}
	}
}

export const themeStore = new ThemeStore();

import type { Config } from './types';

interface ConfigStoreState {
	current: Config | null;
	original: Config | null;
	isLoading: boolean;
	isSaving: boolean;
	error: string;
	saveSuccess: boolean;
	dirtyFields: Set<string>;
}

class ConfigStore {
	private state = $state<ConfigStoreState>({
		current: null,
		original: null,
		isLoading: false,
		isSaving: false,
		error: '',
		saveSuccess: false,
		dirtyFields: new Set()
	});

	// Reactive derived values
	config = $derived(this.state.current);
	isLoading = $derived(this.state.isLoading);
	isSaving = $derived(this.state.isSaving);
	error = $derived(this.state.error);
	saveSuccess = $derived(this.state.saveSuccess);
	dirtyFields = $derived(Array.from(this.state.dirtyFields));
	dirtyFieldCount = $derived(this.state.dirtyFields.size);
	hasChanges = $derived(this.state.dirtyFields.size > 0);

	// Set config from API
	setConfig(config: Config) {
		this.state.current = config;
		this.state.original = this.deepClone(config);
		this.state.dirtyFields = new Set(); // Create new Set to trigger reactivity
	}

	// Update config and track changes
	updateConfig(updater: (config: Config) => void) {
		if (!this.state.current) return;
		
		updater(this.state.current);
		this.detectChanges();
	}

	// Deep clone helper
	private deepClone<T>(obj: T): T {
		return JSON.parse(JSON.stringify(obj));
	}

	// Deep comparison to detect changes
	private detectChanges() {
		if (!this.state.current || !this.state.original) return;

		const dirty = new Set<string>();
		this.compareObjects(this.state.current, this.state.original, '', dirty);
		this.state.dirtyFields = dirty;
	}

	private compareObjects(current: unknown, original: unknown, path: string, dirty: Set<string>) {
		if (current === original) return;

		if (typeof current !== 'object' || current === null || typeof original !== 'object' || original === null) {
			if (current !== original) {
				dirty.add(path);
			}
			return;
		}

		if (Array.isArray(current) && Array.isArray(original)) {
			if (current.length !== original.length) {
				dirty.add(path);
				return;
			}
			for (let i = 0; i < current.length; i++) {
				this.compareObjects(current[i], original[i], `${path}[${i}]`, dirty);
			}
			return;
		}

		const allKeys = new Set([...Object.keys(current as Record<string, unknown>), ...Object.keys(original as Record<string, unknown>)]);
		for (const key of allKeys) {
			const newPath = path ? `${path}.${key}` : key;
			if (!(key in (current as Record<string, unknown>))) {
				dirty.add(newPath);
			} else if (!(key in (original as Record<string, unknown>))) {
				dirty.add(newPath);
			} else {
				this.compareObjects((current as Record<string, unknown>)[key], (original as Record<string, unknown>)[key], newPath, dirty);
			}
		}
	}

	// Cancel changes
	cancel() {
		if (this.state.original) {
			this.state.current = this.deepClone(this.state.original);
			this.state.dirtyFields = new Set(); // Create new Set to trigger reactivity
			this.state.error = '';
		}
	}

	// Export config as JSON
	exportToJSON(): string {
		if (!this.state.current) return '{}';
		return JSON.stringify(this.state.current, null, 2);
	}

	// Import config from JSON
	importFromJSON(jsonString: string): { success: boolean; error?: string } {
		try {
			const parsed = JSON.parse(jsonString);
			
			// Basic validation
			if (!parsed || typeof parsed !== 'object') {
				return { success: false, error: 'Invalid configuration format' };
			}

			// Check for required top-level fields
			const requiredFields = [
				'session_config',
				'network_access_control',
				'proxy_server_config',
				'trusted_proxy_config',
				'portal_user_accounts',
				'protected_services'
			];

			for (const field of requiredFields) {
				if (!(field in parsed)) {
					return { success: false, error: `Missing required field: ${field}` };
				}
			}

			this.state.current = parsed;
			this.detectChanges();
			return { success: true };
		} catch (err) {
			return {
				success: false,
				error: err instanceof Error ? err.message : 'Failed to parse JSON'
			};
		}
	}

	// Copy to clipboard
	async copyToClipboard(): Promise<{ success: boolean; error?: string }> {
		try {
			const json = this.exportToJSON();
			await navigator.clipboard.writeText(json);
			return { success: true };
		} catch (err) {
			return {
				success: false,
				error: err instanceof Error ? err.message : 'Failed to copy to clipboard'
			};
		}
	}

	// Set loading state
	setLoading(loading: boolean) {
		this.state.isLoading = loading;
	}

	// Set saving state
	setSaving(saving: boolean) {
		this.state.isSaving = saving;
	}

	// Set error
	setError(error: string) {
		this.state.error = error;
	}

	// Set save success
	setSaveSuccess(success: boolean) {
		this.state.saveSuccess = success;
		if (success) {
			setTimeout(() => {
				this.state.saveSuccess = false;
			}, 5000);
		}
	}

	// Mark as saved (updates original to current)
	markAsSaved() {
		if (this.state.current) {
			this.state.original = this.deepClone(this.state.current);
			this.state.dirtyFields = new Set(); // Create new Set to trigger reactivity
		}
	}
}

export const configStore = new ConfigStore();

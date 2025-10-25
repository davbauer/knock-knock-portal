import { createToaster } from '@ark-ui/svelte/toast';

export const toaster = createToaster({
	placement: 'top-end',
	overlap: true,
	gap: 16,
	duration: 4000
});

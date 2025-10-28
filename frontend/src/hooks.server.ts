import type { Handle } from '@sveltejs/kit';
import { dev } from '$app/environment';

export const handle: Handle = async ({ event, resolve }) => {
	// Build security headers object
	const securityHeaders: Record<string, string> = {
		// Prevent clickjacking attacks
		'X-Frame-Options': 'DENY',

		// Prevent MIME type sniffing
		'X-Content-Type-Options': 'nosniff',

		// Enable XSS protection (legacy, but still useful for older browsers)
		'X-XSS-Protection': '1; mode=block',

		// Control referrer information
		'Referrer-Policy': 'strict-origin-when-cross-origin',

		// Permissions Policy (formerly Feature Policy)
		'Permissions-Policy':
			'camera=(), microphone=(), geolocation=(), payment=(), usb=(), magnetometer=(), gyroscope=(), accelerometer=()',

		// Content Security Policy - relaxed in dev for backend on different origin
		'Content-Security-Policy': dev
			? [
					"default-src 'self'",
					"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
					"style-src 'self' 'unsafe-inline'",
					"img-src 'self' data: blob:",
					"font-src 'self' data:",
					"connect-src 'self' http://127.0.0.1:8000 http://localhost:8000", // Allow backend API in dev
					"frame-ancestors 'none'",
					"base-uri 'self'",
					"form-action 'self'"
			  ].join('; ')
			: [
					"default-src 'self'",
					"script-src 'self' 'unsafe-inline' 'unsafe-eval'",
					"style-src 'self' 'unsafe-inline'",
					"img-src 'self' data: blob:",
					"font-src 'self' data:",
					"connect-src 'self'", // Strict in production
					"frame-ancestors 'none'",
					"base-uri 'self'",
					"form-action 'self'"
			  ].join('; ')
	};

	// Relax Cross-Origin policies in development
	if (dev) {
		// More permissive in dev for cross-origin API calls
		securityHeaders['Cross-Origin-Embedder-Policy'] = 'unsafe-none';
		securityHeaders['Cross-Origin-Opener-Policy'] = 'unsafe-none';
		securityHeaders['Cross-Origin-Resource-Policy'] = 'cross-origin';
	} else {
		// Strict in production
		securityHeaders['Cross-Origin-Embedder-Policy'] = 'require-corp';
		securityHeaders['Cross-Origin-Opener-Policy'] = 'same-origin';
		securityHeaders['Cross-Origin-Resource-Policy'] = 'same-origin';
	}

	// Custom branding headers
	securityHeaders['X-Powered-By'] = 'Knock-Knock Portal';
	securityHeaders['X-Application'] = 'Knock-Knock Authentication Gateway';
	securityHeaders['X-Version'] = '1.0.0';

	// Cache control for security-sensitive pages
	if (event.url.pathname.startsWith('/admin') || event.url.pathname.startsWith('/portal')) {
		securityHeaders['Cache-Control'] = 'no-store, no-cache, must-revalidate, private';
		securityHeaders['Pragma'] = 'no-cache';
		securityHeaders['Expires'] = '0';
	}

	// Apply security headers using SvelteKit's setHeaders (before resolve)
	event.setHeaders(securityHeaders);

	const response = await resolve(event);

	// Remove potentially sensitive headers
	response.headers.delete('X-SvelteKit-Page');

	return response;
};

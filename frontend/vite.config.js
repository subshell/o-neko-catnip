// vite.config.js
import {resolve} from 'path'
import {defineConfig} from 'vite'

export default defineConfig({
	build: {
		rollupOptions: {
			input: {
				main: resolve(__dirname, 'index.html'),
				wakeup: resolve(__dirname, 'wakeup.html'),
				error: resolve(__dirname, 'error.html'),
			},
		},
	},
	server: {
		proxy: {
			'/api': 'http://localhost:8090'
		}
	}
})

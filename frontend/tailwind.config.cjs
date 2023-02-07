/** @type {import('tailwindcss').Config} */
module.exports = {
	content: [
		"./index.html",
		"./wakeup.html",
		"./error.html",
		"./src/**/*.{js,ts,jsx,tsx}",
	],
	theme: {
		extend: {
			colors: {
				bgdark: {
					800: "#342e2f",
					900: "#252020",
				},
				pink: {
					50 : "#fbe0e8",
					100 : "#f6b3c7",
					200 : "#f080a1",
					300 : "#e94d7b",
					400 : "#e5265f",
					500 : "#e00043",
					600 : "#dc003d",
					700 : "#d80034",
					800 : "#d3002c",
					900 : "#cb001e",
				},
				red: {
					50 : "#fef9f9",
					100 : "#f4b6bb",
					200 : "#ed858e",
					300 : "#e44754",
					400 : "#ff293b",
					500 : "#f91a2d",
					600 : "#d40c1d",
					700 : "#c10313",
					800 : "#a7000f",
					900 : "#8b000b",
				},
				yellow: {
					50 : "#fffbe0",
					100 : "#fff5b3",
					200 : "#ffee80",
					300 : "#ffe74d",
					400 : "#ffe226",
					500 : "#ffdd00",
					600 : "#ffd900",
					700 : "#ffd400",
					800 : "#ffcf00",
					900 : "#ffc700",
				}
			},
			fontFamily: {
				sans: ['Barlow', '-apple-system', 'BlinkMacSystemFont', 'Segoe UI', 'Roboto', 'sans-serif'],
				logo: ['Varela Round', 'Barlow', '-apple-system', 'Roboto', 'sans-serif']
			},
		},
	},
	plugins: [],
}

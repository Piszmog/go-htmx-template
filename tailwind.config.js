/** @type {import('tailwindcss').Config} */
module.exports = {
	content: [
		"./components/**/*.templ",
	],
	theme: {
		extend: {},
	},
	plugins: [
		require('@tailwindcss/forms'),
		require('@tailwindcss/typography'),
	],
}


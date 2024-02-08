/** @type {import('tailwindcss').Config} */
module.exports = {
	content: [
		"./components/**/*.templ",
		"./models/components/**/*.go",
	],
	theme: {
		extend: {},
	},
	plugins: [
		require('@tailwindcss/forms'),
		require('@tailwindcss/typography'),
	],
}


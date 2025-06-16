/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ['../templates/**/*.html', '../static/js/**/*.js'],
  theme: {
    extend: {
      colors: {
        'gollama-primary': '#00d4aa',
        'gollama-secondary': '#1a1a1a',
        'gollama-accent': '#ff6b6b',
      },
      fontFamily: {
        'sans': ['Inter', 'system-ui', 'sans-serif'],
      },
    },
  },
  plugins: [],
  darkMode: 'class',
}

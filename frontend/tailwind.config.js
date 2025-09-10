/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./index.html",
    "./src/**/*.{vue,js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        primary: '#B0D9B1',
        secondary: '#FFD9C0',
        accent: '#E0BBE4',
        background: '#E6E6D8',
        text: '#4A4A4A',
      'dark-purple': '#8B5CF6',
      },
    },
  },
  plugins: [],
}

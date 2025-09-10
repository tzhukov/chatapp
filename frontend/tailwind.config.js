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
        // New darker pastel colors (70-80% of full RGB)
        'sidebar-bg': '#9C7FB8',      // Darker pastel purple
        'chat-bg': '#B8C7A8',        // Darker pastel green
        'input-bg': '#C7A89C',       // Darker pastel brown/tan
        'topbar-bg': '#7F9CB8',      // Darker pastel blue
        'border-subtle': '#6B7280',   // Subtle border color
      },
    },
  },
  plugins: [],
}

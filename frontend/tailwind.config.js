/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{js,jsx}'],
  theme: {
    extend: {
      colors: {
        drive: {
          bg: '#ffffff',
          sidebar: '#f8f9fa',
          border: '#e0e0e0',
          hover: '#f1f3f4',
          active: '#e8f0fe',
          accent: '#1a73e8',
          accentHover: '#1765cc',
          text: '#202124',
          muted: '#5f6368',
          light: '#f1f3f4',
        },
      },
      fontFamily: {
        sans: ['Roboto', 'Arial', 'sans-serif'],
      },
      boxShadow: {
        search: '0 1px 2px 0 rgba(60,64,67,.3), 0 1px 3px 1px rgba(60,64,67,.15)',
        card: '0 1px 2px 0 rgba(60,64,67,.3), 0 1px 3px 1px rgba(60,64,67,.15)',
      },
    },
  },
  plugins: [],
}

/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        military: {
          900: '#0f172a', // Slate 900
          800: '#1e293b', // Slate 800
          700: '#334155', // Slate 700
          accent: '#0ea5e9', // Sky 500
          danger: '#ef4444', // Red 500
          success: '#22c55e', // Green 500
        }
      }
    },
  },
  plugins: [],
}

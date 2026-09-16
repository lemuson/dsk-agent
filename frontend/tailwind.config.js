/** @type {import('tailwindcss').Config} */
export default {
  content: [
    "./index.html",
    "./src/**/*.{js,ts,jsx,tsx}",
  ],
  theme: {
    extend: {
      colors: {
        dsk: {
          50: '#F0F7FF',
          100: '#E0EFFE',
          200: '#B9DDFF',
          300: '#7CBFFF',
          400: '#389DFF',
          500: '#0C7EFF',
          600: '#005FD1',
          700: '#004BAA',
          800: '#03408C',
          900: '#093674',
          950: '#07224B',
        },
        navy: {
          800: '#1E293B',
          900: '#0F172A',
          950: '#0A0F1D',
        }
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', '-apple-system', 'sans-serif'],
      },
      boxShadow: {
        'soft': '0 2px 15px -3px rgba(0, 0, 0, 0.07), 0 10px 20px -2px rgba(0, 0, 0, 0.04)',
        'premium': '0 20px 25px -5px rgba(15, 23, 42, 0.08), 0 8px 10px -6px rgba(15, 23, 42, 0.04)',
      }
    },
  },
  plugins: [],
}

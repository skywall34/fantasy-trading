module.exports = {
  content: [
    './templates/**/*.templ',
    './static/**/*.js',
  ],
  theme: {
    extend: {
      colors: {
        'eog-red': '#E31B23',
        'eog-dark-red': '#B91C1C',
        'eog-black': '#1A1A1A',
      },
      fontFamily: {
        sans: ['Inter', 'Arial', 'sans-serif'],
      },
    },
  },
  plugins: [],
}

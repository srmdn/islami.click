/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./templates/**/*.html"],
  darkMode: "class",
  theme: {
    extend: {
      fontFamily: {
        arabic: ['"Amiri"', '"Scheherazade New"', "serif"],
      },
    },
  },
  plugins: [],
};

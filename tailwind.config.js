/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["./templates/**/*.html"],
  darkMode: "class",
  theme: {
    extend: {
      fontFamily: {
        sans: ['"Plus Jakarta Sans"', 'ui-sans-serif', 'system-ui', 'sans-serif'],
        arabic: ['"Amiri"', '"Scheherazade New"', "serif"],
      },
      boxShadow: {
        "tint-emerald": "0 4px 16px -4px rgba(14, 92, 115, 0.12)",
        "tint-teal": "0 4px 16px -4px rgba(14, 92, 115, 0.10)",
        "card-rest": "0 1px 3px rgba(0,0,0,0.06), 0 1px 2px rgba(0,0,0,0.04)",
        "card-hover": "0 8px 24px -6px rgba(14, 92, 115, 0.10)",
      },
      borderRadius: {
        "2xl": "1rem",
        "3xl": "1.5rem",
      },
    },
  },
  plugins: [],
};

/** @type {import('tailwindcss').Config} */
module.exports = {
  content: ["*.{html,js,php}"],
  theme: {
    extend: {
      "fontSize": {
        "8xl": "24vw",
        "4xl": "12vw",
        "2xl": "6vw"
      }
    },
  },
  plugins: [],
  darkMode: "selector"
}


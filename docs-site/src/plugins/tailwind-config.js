// Docusaurus plugin que registra o PostCSS do Tailwind CSS v4.
//
// Em Tailwind v4 o plugin @tailwindcss/postcss cuida de vendor prefixing e da
// resolucao de @import internamente, entao substituimos a pipeline PostCSS
// default do Docusaurus por apenas o plugin do Tailwind. Isso evita conflitos
// de ordem entre o postcss-preset-env do Docusaurus e o processamento do
// @theme/@custom-variant pelo Tailwind (abordagem documentada por
// https://michal.wrzosek.pl/posts/2025-03-01-tailwind-and-docusaurus).
module.exports = function tailwindPlugin() {
  return {
    name: 'tailwind-plugin',
    configurePostCss(postcssOptions) {
      postcssOptions.plugins = [require('@tailwindcss/postcss')];
      return postcssOptions;
    },
  };
};

const syntaxHighlight = require('@11ty/eleventy-plugin-syntaxhighlight');
const markdownItAnchor = require('markdown-it-anchor');

module.exports = function (eleventyConfig) {
  // Plugins
  eleventyConfig.addPlugin(syntaxHighlight);

  // Markdown config with heading anchors
  eleventyConfig.amendLibrary('md', (mdLib) => {
    mdLib.use(markdownItAnchor, {
      permalink: markdownItAnchor.permalink.headerLink(),
      level: [2, 3, 4],
    });
  });

  // Passthrough copy
  eleventyConfig.addPassthroughCopy({ 'src/assets': 'assets' });

  // Collections
  eleventyConfig.addCollection('docs', function (collectionApi) {
    return collectionApi
      .getFilteredByTag('docs')
      .sort((a, b) => (a.data.order || 0) - (b.data.order || 0));
  });

  eleventyConfig.addCollection('posts', function (collectionApi) {
    return collectionApi.getFilteredByTag('posts').sort((a, b) => b.date - a.date);
  });

  // Filters
  eleventyConfig.addFilter('dateFormat', function (date) {
    return new Date(date).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  });

  eleventyConfig.addFilter('year', function () {
    return new Date().getFullYear();
  });

  return {
    dir: {
      input: 'src',
      output: '_site',
    },
  };
};

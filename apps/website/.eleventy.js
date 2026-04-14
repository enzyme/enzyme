const syntaxHighlight = require('@11ty/eleventy-plugin-syntaxhighlight');
const markdownItAnchor = require('markdown-it-anchor');

module.exports = function (eleventyConfig) {
  // Plugins
  eleventyConfig.addPlugin(syntaxHighlight);

  // Markdown config with heading anchors
  eleventyConfig.amendLibrary('md', (mdLib) => {
    mdLib.use(markdownItAnchor, {
      permalink: markdownItAnchor.permalink.ariaHidden({
        placement: 'after',
        symbol: '#',
        class: 'header-anchor',
      }),
      level: [2, 3, 4],
    });
  });

  // Don't inherit .gitignore (src/docs/api/ is gitignored but needed by Eleventy)
  eleventyConfig.setUseGitIgnore(false);

  // Passthrough copy
  eleventyConfig.addPassthroughCopy({ 'src/assets': 'assets' });

  // Collections
  const DOC_SECTIONS = [
    'Getting Started',
    'Using Enzyme',
    'Administration',
    'Self-Hosting & Operations',
    'API Reference',
  ];

  eleventyConfig.addCollection('docs', function (collectionApi) {
    return collectionApi
      .getFilteredByTag('docs')
      .sort((a, b) => (a.data.order || 0) - (b.data.order || 0));
  });

  eleventyConfig.addCollection('docSections', function (collectionApi) {
    const docs = collectionApi
      .getFilteredByTag('docs')
      .sort((a, b) => (a.data.order || 0) - (b.data.order || 0));

    return DOC_SECTIONS.map((name) => ({
      name,
      docs: docs.filter((doc) => doc.data.section === name),
    })).filter((section) => section.docs.length > 0);
  });

  eleventyConfig.addCollection('posts', function (collectionApi) {
    return collectionApi
      .getFilteredByTag('posts')
      .sort((a, b) => b.date.getTime() - a.date.getTime());
  });

  eleventyConfig.addCollection('comparisons', function (collectionApi) {
    return collectionApi
      .getFilteredByTag('comparisons')
      .sort((a, b) => (a.data.order || 0) - (b.data.order || 0));
  });

  // Filters
  eleventyConfig.addFilter('dateFormat', function (date) {
    var d = new Date(date);
    if (isNaN(d.getTime())) {
      throw new Error('dateFormat filter received invalid date: ' + date);
    }
    return d.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  });

  eleventyConfig.addFilter('year', function () {
    return new Date().getFullYear();
  });

  eleventyConfig.addFilter('isoDate', function (date) {
    return new Date(date).toISOString();
  });

  return {
    dir: {
      input: 'src',
      output: '_site',
    },
  };
};

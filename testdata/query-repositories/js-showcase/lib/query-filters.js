const queryFilters = function(filters) {
  const frameworks = filters.framework || [];
  // Framed UTF-8 hex preserves exact membership for a single bound SQL value.
  const keys = frameworks.map(value => encodeURIComponent(value)
    .replace(/%[0-9A-F]{2}|./g, token => token[0] === "%"
      ? token.slice(1)
      : token.charCodeAt(0).toString(16).padStart(2, "0").toUpperCase()));
  return {
    frameworkKeys: frameworks.length ? "|" + keys.join("|") + "|" : null,
    cwd: filters.working_directory_like ? "%" + filters.working_directory_like + "%" : null,
    minTools: filters.min_tool_calls || null,
  };
};

exports.queryFilters = queryFilters;

__section__("filters", {
  title: "Filters",
  fields: {
    source: { type: "string", help: "Source path" },
  },
});
function good(filters) { return [{ source: filters.source }]; }
__verb__("good", { name: "good", short: "Good", fields: { filters: { bind: "filters" } } });

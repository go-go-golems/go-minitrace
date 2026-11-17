__section__("bad_filters", {
  title: "Filters",
  fields: {
    source: {
      type: "string",
      help: "Normalized archive file path",
    },
    offset: {
      type: "int",
      default: -1,
      help: "Signed default sentinel",
    },
  },
});

function bad(filters) {
  return [{ source: filters.source ?? null, offset: filters.offset }];
}

__verb__("bad", {
  name: "bad",
  short: "Bad sibling with a negative numeric metadata default",
  fields: { bad_filters: { bind: "bad_filters" } },
});

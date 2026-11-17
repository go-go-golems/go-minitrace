function bad(inputs) {
  return [{ offset: inputs ? inputs.offset : null }];
}

__verb__("bad", {
  name: "bad",
  short: "Signed numeric default probe sibling",
  fields: {
    offset: {
      type: "int",
      default: -1,
      help: "Signed numeric default in JS __verb__ metadata",
    },
  },
});

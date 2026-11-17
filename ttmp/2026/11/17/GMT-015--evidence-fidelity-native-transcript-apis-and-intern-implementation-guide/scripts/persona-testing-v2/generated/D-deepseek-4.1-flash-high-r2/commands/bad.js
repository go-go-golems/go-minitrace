function bad(delta) {
  return [{ delta: delta }];
}

__verb__("bad", {
  name: "bad",
  short: "Bad sibling carrying a negative numeric metadata default",
  fields: {
    delta: { type: "int", default: -1, help: "Signed numeric metadata default probe" },
  },
});

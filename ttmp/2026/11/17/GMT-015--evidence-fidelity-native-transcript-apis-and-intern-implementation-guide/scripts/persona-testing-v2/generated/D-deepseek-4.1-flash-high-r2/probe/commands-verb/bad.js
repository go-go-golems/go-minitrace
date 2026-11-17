function bad(delta) {
  return [{ delta: delta }];
}

__verb__("bad", {
  name: "bad",
  short: "Verb metadata negative default probe",
  fields: {
    delta: { type: "int", default: -1, help: "Signed numeric default" },
  },
});

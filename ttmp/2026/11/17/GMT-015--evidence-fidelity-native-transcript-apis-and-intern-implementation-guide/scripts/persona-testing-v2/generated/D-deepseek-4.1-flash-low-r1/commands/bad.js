function bad(offset) {
  return [{ offset: offset }];
}

__verb__("bad", { name: "bad", short: "Bad sibling with a negative numeric metadata default", params: [{ name: "offset", type: "int", default: -1, bind: "offset", help: "Signed default sentinel" }] });

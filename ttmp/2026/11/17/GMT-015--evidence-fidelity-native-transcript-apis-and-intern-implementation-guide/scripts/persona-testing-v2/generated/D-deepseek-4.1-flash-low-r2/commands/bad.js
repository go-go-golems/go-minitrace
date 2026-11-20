__section__("settings", {
  title: "Settings",
  fields: {
    limit: {
      type: "int",
      help: "Signed default probe",
      default: -1,
    },
  },
});

function bad(settings) {
  return [{ signed_default: settings.limit }];
}

__verb__("bad", {
  name: "bad",
  short: "Signed numeric default probe",
  fields: { settings: { bind: "settings" } },
});

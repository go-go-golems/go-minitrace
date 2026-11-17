// Compare CLI paths: query commands runtime-archives-helper inline / helper
function withDatabase(mt, callback) {
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try { return callback(db); } finally { db.close(); }
}
function helper() {
  const mt = require('minitrace');
  return withDatabase(mt, db => db.query('SELECT COUNT(*) AS sessions FROM sessions'));
}

__verb__('helper', {name:'helper',short:'Same builder called through helper'});

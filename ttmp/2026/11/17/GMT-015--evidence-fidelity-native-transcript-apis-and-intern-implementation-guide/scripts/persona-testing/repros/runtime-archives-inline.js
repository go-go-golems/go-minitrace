function inline() {
  const mt = require('minitrace');
  const db = mt.db().RuntimeArchives().SQLiteMemory().Build();
  try { return db.query('SELECT COUNT(*) AS sessions FROM sessions'); }
  finally { db.close(); }
}

__verb__('inline', {name:'inline',short:'Inline control'});

// Synthetic evidence only; no historical archives or external services required.
function probe() {
  const mt = require('minitrace');
  const source = JSON.stringify({id:'fidelity-probe',profile:'organic',turns:[],tool_calls:[]});
  const output = [];
  [false, true].forEach(function(strict) {
    const db = mt.db().Content(source).SQLiteMemory().MaxCellBytes(3)
      .MaxRows(1).StrictEvidence(strict).Build();
    try {
      const result = db.execute({
        sql: 'SELECT ? AS evidence UNION ALL SELECT ? ORDER BY evidence',
        args: ['abé', 'second']
      });
      if(result.complete || !result.truncated || !result.rowsTruncated)
        throw new Error('Expected explicit incompleteness');
      if(result.cellDiagnostics[0].originalBytes !== 4 || result.cellDiagnostics[0].emittedBytes !== 2)
        throw new Error('Expected valid UTF-8 boundary diagnostic');
      if(strict && (!result.error || result.rows.length !== 0))
        throw new Error('Strict mode leaked incomplete rows');
      if(!strict && result.rows[0].evidence !== 'ab')
        throw new Error('Permissive mode split UTF-8');
      output.push({strict:strict, result:JSON.stringify(result)});
    } finally { db.close(); }
  });
  return output;
}
__verb__('probe', {name:'fidelity-probe',short:'Synthetic UTF-8, row overflow and strict evidence regression'});

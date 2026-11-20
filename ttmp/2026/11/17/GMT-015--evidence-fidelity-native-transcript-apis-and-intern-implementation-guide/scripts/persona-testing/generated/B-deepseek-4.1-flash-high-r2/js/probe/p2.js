__section__("opts",{title:"o",fields:{a:{type:"string",default:"x",help:"h"}}});
function s(opts){const mt=require("minitrace");
const db=mt.db().RuntimeArchives().SQLiteMemory().Build();
try{const t=db.tables();const turns=t.filter(x=>x.name==="turns")[0];
const r=mt.query().TokenUsageRows().Build();
return {turnCols:turns.columns.map(c=>c.name), recipeSql:r.sql(), recipeArgs:r.args(), rows:db.execute(r).rows};}finally{db.close();}}
__verb__("s",{name:"s",short:"s",fields:{opts:{bind:"opts"}}});

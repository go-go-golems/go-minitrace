__section__("opts",{title:"o",fields:{a:{type:"string",default:"x",help:"h"}}});
function s(opts){const mt=require("minitrace");
const db=mt.db().RuntimeArchives().SQLiteMemory().Build();
try{const t=db.tables();const out={};
for(const tb of t){out[tb.name]=tb.columns.map(c=>c.name).filter(n=>/token|origin|method|unit/.test(n));}
return out;}finally{db.close();}}
__verb__("s",{name:"s",short:"s",fields:{opts:{bind:"opts"}}});

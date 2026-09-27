import ts from "/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/app/node_modules/typescript/lib/typescript.js";
import fs from "node:fs"; import path from "node:path";
const base=path.dirname(new URL(import.meta.url).pathname), root="/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/app";
const config=ts.readConfigFile(path.join(root,"tsconfig.app.json"),ts.sys.readFile);
const parsed=ts.parseJsonConfigFileContent(config.config,ts.sys,root);
const options={...parsed.options,noEmit:false,removeComments:true,sourceMap:false,inlineSourceMap:false};
const name=path.join(root,"codegen/gotypes.gen.ts");
function emit(text){
 const host=ts.createCompilerHost(options), original=host.readFile.bind(host); host.readFile=p=>path.resolve(p)===name?text:original(p);
 const program=ts.createProgram([name],options,host); let js="";
 const diagnostics=ts.getPreEmitDiagnostics(program);
 const result=program.emit(program.getSourceFile(name),(p,value)=>{if(p.endsWith("gotypes.gen.js"))js=value;});
 if(result.emitSkipped||!js||diagnostics.length||result.diagnostics.length)throw new Error(ts.formatDiagnostics([...diagnostics,...result.diagnostics],{getCurrentDirectory:()=>root,getNewLine:()=>"\n",getCanonicalFileName:x=>x}));
 return js;
}
const raw=fs.readFileSync(path.join(base,"baseline/gotypes.raw.ts"),"utf8"),candidate=fs.readFileSync(path.join(base,"gotypes.candidate.ts"),"utf8");
const a=emit(raw),b=emit(candidate);fs.writeFileSync(path.join(base,"program-before.js"),a);fs.writeFileSync(path.join(base,"program-after.js"),b);
console.log(JSON.stringify({compiler:ts.version,options,identical:a===b,bytes:Buffer.byteLength(a),boundary:"Real TypeScript Program for generated module using parsed project options; no browser/Vite runtime claim"},null,2));
if(a!==b)process.exitCode=1;

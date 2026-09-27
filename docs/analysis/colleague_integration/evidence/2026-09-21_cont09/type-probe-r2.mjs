// Design experiment only: no repository-source writes.
import ts from '/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/app/node_modules/typescript/lib/typescript.js';
import fs from 'node:fs';
import path from 'node:path';
const root = '/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/app';
const here = path.dirname(new URL(import.meta.url).pathname);
const raw = fs.readFileSync(path.join(here, 'baseline/gotypes.raw.ts'), 'utf8');
const sf = ts.createSourceFile('gotypes.gen.ts', raw, ts.ScriptTarget.Latest, true);
const parseType = text => ts.createSourceFile('type.ts', `type T = ${text};`, ts.ScriptTarget.Latest, true).statements[0].type;
const edits = [];
const change = (node, text) => edits.push({start: node.getStart(sf), end: node.end, text});
let classes = 0, helpers = 0;
for (const cls of sf.statements) {
  if (!ts.isClassDeclaration(cls)) throw new Error('Unexpected top-level statement');
  classes++;
  for (const member of cls.members) {
    if (ts.isPropertyDeclaration(member)) {
      const visit = node => {
        if (node.kind === ts.SyntaxKind.AnyKeyword) change(node, 'JSONValue');
        ts.forEachChild(node, visit);
      };
      visit(member.type);
    } else if (ts.isConstructorDeclaration(member)) {
      change(member.parameters[0].type, 'unknown');
      for (const statement of member.body.statements) {
        if (!ts.isExpressionStatement(statement) || !ts.isBinaryExpression(statement.expression)) continue;
        const assignment = statement.expression;
        if (!ts.isPropertyAccessExpression(assignment.left) || assignment.left.expression.kind !== ts.SyntaxKind.ThisKeyword) throw new Error('Unknown constructor write');
        const target = `${cls.name.text}[${JSON.stringify(assignment.left.name.text)}]`;
        const rhs = assignment.right;
        const local = [];
        const visit = node => {
          if (ts.isElementAccessExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === 'source') {
            local.push({start: node.expression.getStart(sf), end: node.expression.end, text: '(source as Record<string, unknown>)'});
            if (ts.isNewExpression(node.parent) && node.parent.expression.getText(sf) === 'Date') {
              local.push({start: node.end, end: node.end, text: ' as string | number | Date'});
            }
          }
          ts.forEachChild(node, visit);
        };
        visit(rhs);
        let text = raw.slice(rhs.getStart(sf), rhs.end);
        for (const edit of local.sort((a,b) => b.start-a.start)) {
          const a=edit.start-rhs.getStart(sf), b=edit.end-rhs.getStart(sf);
          text=text.slice(0,a)+edit.text+text.slice(b);
        }
        if (ts.isBinaryExpression(rhs)) {
          if (rhs.operatorToken.kind !== ts.SyntaxKind.AmpersandAmpersandToken || !ts.isElementAccessExpression(rhs.left) || !ts.isNewExpression(rhs.right) || rhs.right.expression.getText(sf) !== 'Date') throw new Error('Unsupported transform expression');
          const boundary = text.indexOf(' && ');
          text = text.slice(0, boundary) + ` as ${target}` + text.slice(boundary);
          change(rhs, text);
        } else {
          change(rhs, `${text} as ${target}`);
        }
      }
    } else if (ts.isMethodDeclaration(member) && member.name.getText(sf) === 'convertValues') {
      helpers++;
      change(member.parameters[0].type, 'unknown');
      change(member.parameters[1].type, 'new (source: unknown) => unknown');
      change(member.type, 'unknown');
      const visit = node => {
        if (node.kind === ts.SyntaxKind.AnyKeyword) change(node, 'unknown');
        if (ts.isElementAccessExpression(node) && ts.isIdentifier(node.expression) && node.expression.text === 'a') change(node.expression, '(a as Record<string, unknown>)');
        ts.forEachChild(node, visit);
      };
      visit(member.body);
    } else throw new Error('Unexpected class member');
  }
}
let result=raw;
for(const e of edits.sort((a,b)=>b.start-a.start)) result=result.slice(0,e.start)+e.text+result.slice(e.end);
result='type JSONValue = string | number | boolean | null | JSONValue[] | { [key: string]: JSONValue };\n\n'+result;
fs.writeFileSync(path.join(here,'gotypes.candidate.ts'),result);
const config = ts.readConfigFile(path.join(root,'tsconfig.app.json'),ts.sys.readFile);
const parsed=ts.parseJsonConfigFileContent(config.config,ts.sys,root);
const host=ts.createCompilerHost(parsed.options),original=host.readFile.bind(host);
host.readFile=filename=>path.resolve(filename)===path.join(root,'codegen/gotypes.gen.ts')?result:original(filename);
const program=ts.createProgram(parsed.fileNames,parsed.options,host);
const diagnostics=ts.getPreEmitDiagnostics(program).map(d=>({file:d.file?.fileName,line:d.file&&d.start!=null?d.file.getLineAndCharacterOfPosition(d.start).line+1:null,message:ts.flattenDiagnosticMessageText(d.messageText,'\n')}));
const opts={target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.ESNext,useDefineForClassFields:true,removeComments:true};
const before=ts.transpileModule(raw,{compilerOptions:opts}).outputText;
const after=ts.transpileModule(result,{compilerOptions:opts}).outputText;
fs.writeFileSync(path.join(here,'before.js'),before);fs.writeFileSync(path.join(here,'after.js'),after);
console.log(JSON.stringify({classes,helpers,diagnostics,emitted_bytes_identical:before===after},null,2));

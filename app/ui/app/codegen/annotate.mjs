import ts from "typescript";

export const jsonType = "export type JSONValue = string | number | boolean | null | JSONValue[] | { [key: string]: JSONValue };\n\n";
const parse = text => ts.createSourceFile("gotypes.gen.ts", text, ts.ScriptTarget.Latest, true);
const printer = ts.createPrinter({ removeComments: true });
const canonical = (node, file) => printer.printNode(ts.EmitHint.Unspecified, node, file);
const reference = parse(`export class Reference {
 constructor(source: any = {}) {
  if ('string' === typeof source) source = JSON.parse(source);
 }
 convertValues(a: any, classs: any, asMap: boolean = false): any {
  if (!a) { return a; }
  if (Array.isArray(a)) {
   return (a as any[]).map(elem => this.convertValues(elem, classs));
  } else if ("object" === typeof a) {
   if (asMap) {
    for (const key of Object.keys(a)) { a[key] = new classs(a[key]); }
    return a;
   }
   return new classs(a);
  }
  return a;
 }
}`);
const [referenceConstructor, referenceHelper] = reference.statements[0].members;
const requireShape = (condition, message) => {
  if (!condition) throw new Error(`Unsupported generated shape: ${message}`);
};
const identifier = (node, name) => ts.isIdentifier(node) && node.text === name;
function walk(node, visit) {
  visit(node);
  ts.forEachChild(node, child => walk(child, visit));
}

// Types refine the existing unchecked materialisation boundary; they do not validate input.
export function annotate(raw) {
  const file = parse(raw);
  requireShape(file.parseDiagnostics.length === 0, "parse failure");
  requireShape(file.statements.length > 0, "empty module");
  const names = new Set();
  for (const cls of file.statements) {
    requireShape(ts.isClassDeclaration(cls) && cls.name, "only named classes are supported");
    requireShape(cls.name.text !== "JSONValue", "JSONValue name collision");
    requireShape(!names.has(cls.name.text), "duplicate class");
    names.add(cls.name.text);
    requireShape(!cls.heritageClauses && !cls.typeParameters && cls.modifiers?.length === 1 &&
      cls.modifiers[0].kind === ts.SyntaxKind.ExportKeyword, "class modifiers");
  }
  const edits = [];
  const replace = (node, text) => edits.push({ start: node.getStart(file), end: node.end, text });
  const insert = (offset, text) => edits.push({ start: offset, end: offset, text });
  let helpers = 0;
  for (const cls of file.statements) {
    const properties = new Set();
    for (const member of cls.members.filter(ts.isPropertyDeclaration)) {
      requireShape(ts.isIdentifier(member.name) && member.type && !member.initializer &&
        !member.modifiers && !member.exclamationToken, "public field declaration");
      requireShape(!properties.has(member.name.text), "duplicate public field");
      properties.add(member.name.text);
      walk(member.type, node => requireShape(node.kind !== ts.SyntaxKind.AnyKeyword,
        `public field ${cls.name.text}.${member.name.text} requires a Go-owned ts_type`));
    }
    let constructors = 0, classHelpers = 0;
    for (const member of cls.members) {
      if (ts.isPropertyDeclaration(member)) continue;
      if (ts.isConstructorDeclaration(member)) {
        constructors++;
        requireShape(!member.modifiers && member.parameters.length === 1 && member.body, "constructor signature");
        requireShape(canonical(member.parameters[0], file) === canonical(referenceConstructor.parameters[0], reference), "constructor parameter");
        const statements = member.body.statements;
        requireShape(statements.length === properties.size + 1 &&
          canonical(statements[0], file) === canonical(referenceConstructor.body.statements[0], reference), "JSON constructor preamble or field count");
        replace(member.parameters[0].type, "unknown");
        const assigned = new Set();
        for (const statement of statements.slice(1)) {
          requireShape(ts.isExpressionStatement(statement) && ts.isBinaryExpression(statement.expression), "constructor statement");
          const assignment = statement.expression;
          requireShape(assignment.operatorToken.kind === ts.SyntaxKind.EqualsToken &&
            ts.isPropertyAccessExpression(assignment.left) && assignment.left.expression.kind === ts.SyntaxKind.ThisKeyword,
            "constructor assignment");
          const field = assignment.left.name.text;
          requireShape(properties.has(field) && !assigned.has(field), "unknown or repeated field write");
          assigned.add(field);
          const target = `${cls.name.text}[${JSON.stringify(field)}]`;
          const read = node => {
            requireShape(ts.isElementAccessExpression(node) && identifier(node.expression, "source") &&
              ts.isStringLiteral(node.argumentExpression) && node.argumentExpression.text === field, "source field read");
            replace(node.expression, "(source as Record<string, unknown>)");
          };
          const date = node => {
            requireShape(ts.isNewExpression(node) && identifier(node.expression, "Date") &&
              !node.typeArguments && node.arguments?.length === 1, "Date transform");
            read(node.arguments[0]);
            insert(node.arguments[0].end, " as string | number | Date");
          };
          const rhs = assignment.right;
          if (ts.isElementAccessExpression(rhs)) read(rhs);
          else if (ts.isNewExpression(rhs)) date(rhs);
          else if (ts.isCallExpression(rhs)) {
            requireShape(ts.isPropertyAccessExpression(rhs.expression) &&
              rhs.expression.expression.kind === ts.SyntaxKind.ThisKeyword && identifier(rhs.expression.name, "convertValues") &&
              !rhs.typeArguments && [2, 3].includes(rhs.arguments.length) &&
              ts.isIdentifier(rhs.arguments[1]) && names.has(rhs.arguments[1].text) &&
              (rhs.arguments.length === 2 || rhs.arguments[2].kind === ts.SyntaxKind.TrueKeyword), "nested conversion");
            read(rhs.arguments[0]);
          } else if (ts.isBinaryExpression(rhs)) {
            requireShape(rhs.operatorToken.kind === ts.SyntaxKind.AmpersandAmpersandToken, "guard operator");
            read(rhs.left);
            date(rhs.right);
            insert(rhs.left.end, ` as ${target}`);
            continue;
          } else requireShape(false, "constructor transform expression");
          insert(rhs.end, ` as ${target}`);
        }
      } else if (ts.isMethodDeclaration(member)) {
        requireShape(canonical(member, file) === canonical(referenceHelper, reference), "convertValues template");
        classHelpers++;
        helpers++;
        replace(member.parameters[0].type, "unknown");
        replace(member.parameters[1].type, "new (source: unknown) => unknown");
        replace(member.type, "unknown");
        walk(member.body, node => {
          if (node.kind === ts.SyntaxKind.AnyKeyword) replace(node, "unknown");
          if (ts.isElementAccessExpression(node) && identifier(node.expression, "a"))
            replace(node.expression, "(a as Record<string, unknown>)");
        });
      } else requireShape(false, "class member");
    }
    requireShape(constructors === 1 && classHelpers <= 1, "constructor/helper count");
  }
  edits.sort((a, b) => a.start - b.start || a.end - b.end);
  let end = 0, output = "";
  for (const edit of edits) {
    requireShape(edit.start >= end && edit.end >= edit.start, "overlapping annotation edits");
    output += raw.slice(end, edit.start) + edit.text;
    end = edit.end;
  }
  output = jsonType + output + raw.slice(end);
  const result = parse(output);
  requireShape(result.parseDiagnostics.length === 0, "annotation parse failure");
  walk(result, node => requireShape(node.kind !== ts.SyntaxKind.AnyKeyword, "unhandled explicit-any"));
  return { text: output, classes: names.size, helpers };
}

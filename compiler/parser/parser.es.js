function peg$subclass(child, parent) {
  function ctor() { this.constructor = child; }
  ctor.prototype = parent.prototype;
  child.prototype = new ctor();
}

function peg$SyntaxError(message, expected, found, location) {
  this.message  = message;
  this.expected = expected;
  this.found    = found;
  this.location = location;
  this.name     = "SyntaxError";

  if (typeof Error.captureStackTrace === "function") {
    Error.captureStackTrace(this, peg$SyntaxError);
  }
}

peg$subclass(peg$SyntaxError, Error);

peg$SyntaxError.buildMessage = function(expected, found) {
  var DESCRIBE_EXPECTATION_FNS = {
        literal: function(expectation) {
          return "\"" + literalEscape(expectation.text) + "\"";
        },

        "class": function(expectation) {
          var escapedParts = "",
              i;

          for (i = 0; i < expectation.parts.length; i++) {
            escapedParts += expectation.parts[i] instanceof Array
              ? classEscape(expectation.parts[i][0]) + "-" + classEscape(expectation.parts[i][1])
              : classEscape(expectation.parts[i]);
          }

          return "[" + (expectation.inverted ? "^" : "") + escapedParts + "]";
        },

        any: function(expectation) {
          return "any character";
        },

        end: function(expectation) {
          return "end of input";
        },

        other: function(expectation) {
          return expectation.description;
        }
      };

  function hex(ch) {
    return ch.charCodeAt(0).toString(16).toUpperCase();
  }

  function literalEscape(s) {
    return s
      .replace(/\\/g, '\\\\')
      .replace(/"/g,  '\\"')
      .replace(/\0/g, '\\0')
      .replace(/\t/g, '\\t')
      .replace(/\n/g, '\\n')
      .replace(/\r/g, '\\r')
      .replace(/[\x00-\x0F]/g,          function(ch) { return '\\x0' + hex(ch); })
      .replace(/[\x10-\x1F\x7F-\x9F]/g, function(ch) { return '\\x'  + hex(ch); });
  }

  function classEscape(s) {
    return s
      .replace(/\\/g, '\\\\')
      .replace(/\]/g, '\\]')
      .replace(/\^/g, '\\^')
      .replace(/-/g,  '\\-')
      .replace(/\0/g, '\\0')
      .replace(/\t/g, '\\t')
      .replace(/\n/g, '\\n')
      .replace(/\r/g, '\\r')
      .replace(/[\x00-\x0F]/g,          function(ch) { return '\\x0' + hex(ch); })
      .replace(/[\x10-\x1F\x7F-\x9F]/g, function(ch) { return '\\x'  + hex(ch); });
  }

  function describeExpectation(expectation) {
    return DESCRIBE_EXPECTATION_FNS[expectation.type](expectation);
  }

  function describeExpected(expected) {
    var descriptions = new Array(expected.length),
        i, j;

    for (i = 0; i < expected.length; i++) {
      descriptions[i] = describeExpectation(expected[i]);
    }

    descriptions.sort();

    if (descriptions.length > 0) {
      for (i = 1, j = 1; i < descriptions.length; i++) {
        if (descriptions[i - 1] !== descriptions[i]) {
          descriptions[j] = descriptions[i];
          j++;
        }
      }
      descriptions.length = j;
    }

    switch (descriptions.length) {
      case 1:
        return descriptions[0];

      case 2:
        return descriptions[0] + " or " + descriptions[1];

      default:
        return descriptions.slice(0, -1).join(", ")
          + ", or "
          + descriptions[descriptions.length - 1];
    }
  }

  function describeFound(found) {
    return found ? "\"" + literalEscape(found) + "\"" : "end of input";
  }

  return "Expected " + describeExpected(expected) + " but " + describeFound(found) + " found.";
};

function peg$parse(input, options) {
  options = options !== void 0 ? options : {};

  var peg$FAILED = {},

      peg$startRuleFunctions = { start: peg$parsestart, Expr: peg$parseExpr },
      peg$startRuleFunction  = peg$parsestart,

      peg$c0 = function(scope) { return [scope] },
      peg$c1 = function(seq) { return seq },
      peg$c2 = function(decls, body) {
            return {"kind": "Scope", "decls": decls, "body": body}
          },
      peg$c3 = function(first, rest) {
            return [first, ... rest]
          },
      peg$c4 = function(o) { return o },
      peg$c5 = function(v) { return v },
      peg$c6 = "const",
      peg$c7 = peg$literalExpectation("const", false),
      peg$c8 = "=",
      peg$c9 = peg$literalExpectation("=", false),
      peg$c10 = function(id, expr) {
            return {"kind": "ConstDecl", "name": id, "expr": expr}
          },
      peg$c11 = "type",
      peg$c12 = peg$literalExpectation("type", false),
      peg$c13 = function(id, typ) {
            return {
              
            "kind": "ConstDecl",
              
            "name": id,
              
            "expr": {"kind": "TypeValue", "value": {"kind": "TypeDef", "name": id, "type": typ}}}
          
          },
      peg$c14 = "func",
      peg$c15 = peg$literalExpectation("func", false),
      peg$c16 = "(",
      peg$c17 = peg$literalExpectation("(", false),
      peg$c18 = ")",
      peg$c19 = peg$literalExpectation(")", false),
      peg$c20 = ":",
      peg$c21 = peg$literalExpectation(":", false),
      peg$c22 = function(id, params, expr) {
            return {
              
            "kind": "FuncDecl",
              
            "name": id,
              
            "params": params,
              
            "expr": expr}
          
          },
      peg$c23 = "op",
      peg$c24 = peg$literalExpectation("op", false),
      peg$c25 = function(name, params, body) {
            return {
              
            "kind": "OpDecl",
              
            "name": name,
              
            "params": params,
              
            "body": body}
          
          },
      peg$c26 = ",",
      peg$c27 = peg$literalExpectation(",", false),
      peg$c28 = function(first, p) { return p },
      peg$c29 = "...",
      peg$c30 = peg$literalExpectation("...", false),
      peg$c31 = function() { return {"kind": "SpreadParam"} },
      peg$c32 = function(id) { return {"kind": "NamedParam", "name": id} },
      peg$c33 = "fork",
      peg$c34 = peg$literalExpectation("fork", false),
      peg$c35 = function(paths) {
            return {"kind": "Parallel", "paths": paths}
          },
      peg$c36 = "switch",
      peg$c37 = peg$literalExpectation("switch", false),
      peg$c38 = function(expr, cases) {
            return {"kind": "Switch", "expr": expr, "cases": cases}
          },
      peg$c39 = function(cases) {
            return {"kind": "Switch", "expr": null, "cases": cases}
          },
      peg$c40 = "from",
      peg$c41 = peg$literalExpectation("from", false),
      peg$c42 = function(trunks) {
            return {"kind": "From", "trunks": trunks}
          },
      peg$c43 = function(scope) { return scope },
      peg$c44 = function(a) { return a },
      peg$c45 = "search",
      peg$c46 = peg$literalExpectation("search", false),
      peg$c47 = function(expr) {
            return {"kind": "Search", "expr": expr}
          },
      peg$c48 = function(expr) {
            return {"kind": "OpExpr", "expr": expr}
          },
      peg$c49 = function(expr) {
            return {"kind": "OpExpr", "expr": expr}
        },
      peg$c50 = "=>",
      peg$c51 = peg$literalExpectation("=>", false),
      peg$c52 = "|",
      peg$c53 = peg$literalExpectation("|", false),
      peg$c54 = "{",
      peg$c55 = peg$literalExpectation("{", false),
      peg$c56 = "[",
      peg$c57 = peg$literalExpectation("[", false),
      peg$c58 = function(expr, path) {
            return {"expr": expr, "path": path}
          },
      peg$c59 = "case",
      peg$c60 = peg$literalExpectation("case", false),
      peg$c61 = function(expr) { return expr },
      peg$c62 = "default",
      peg$c63 = peg$literalExpectation("default", false),
      peg$c64 = function() { return null },
      peg$c65 = function(source, opt) {
            let m = {"kind": "Trunk", "source": source, "seq": null};
            if (opt) {
              m["seq"] = opt[3];
            }
            return m
          },
      peg$c66 = "~",
      peg$c67 = peg$literalExpectation("~", false),
      peg$c68 = "==",
      peg$c69 = peg$literalExpectation("==", false),
      peg$c70 = "!=",
      peg$c71 = peg$literalExpectation("!=", false),
      peg$c72 = "in",
      peg$c73 = peg$literalExpectation("in", false),
      peg$c74 = "<=",
      peg$c75 = peg$literalExpectation("<=", false),
      peg$c76 = "<",
      peg$c77 = peg$literalExpectation("<", false),
      peg$c78 = ">=",
      peg$c79 = peg$literalExpectation(">=", false),
      peg$c80 = ">",
      peg$c81 = peg$literalExpectation(">", false),
      peg$c82 = function() { return text() },
      peg$c83 = function(first, rest) {
            return makeBinaryExprChain(first, rest)
          },
      peg$c84 = function(t) { return ["or", t] },
      peg$c85 = function(first, expr) { return ["and", expr] },
      peg$c86 = "!",
      peg$c87 = peg$literalExpectation("!", false),
      peg$c88 = function(e) {
            return {"kind": "UnaryExpr", "op": "!", "operand": e}
          },
      peg$c89 = function(v) {
            return {"kind": "Term", "text": text(), "value": v}
          },
      peg$c90 = "*",
      peg$c91 = peg$literalExpectation("*", false),
      peg$c92 = function() {
            return {"kind": "Primitive", "type": "bool", "text": "true"}
          },
      peg$c93 = function(lhs, op, rhs) {
            return {"kind": "BinaryExpr", "op": op, "lhs": lhs, "rhs": rhs}
          },
      peg$c94 = function(first, rest) {
               return makeBinaryExprChain(first, rest)
           },
      peg$c95 = function(v) {
            return {"kind": "Primitive", "type": "string", "text": v}
          },
      peg$c96 = function(pattern) {
            return {"kind": "Glob", "pattern": pattern}
        },
      peg$c97 = function(pattern) {
            return {"kind": "Regexp", "pattern": pattern}
        },
      peg$c98 = function(keys, limit) {
            return {"kind": "Summarize", "keys": keys, "aggs": null, "limit": limit}
          },
      peg$c99 = function(aggs, keys, limit) {
            let p = {"kind": "Summarize", "keys": null, "aggs": aggs, "limit": limit};
            if (keys) {
              p["keys"] = keys[1];
            }
            return p
          },
      peg$c100 = "summarize",
      peg$c101 = peg$literalExpectation("summarize", false),
      peg$c102 = function(columns) { return columns },
      peg$c103 = "with",
      peg$c104 = peg$literalExpectation("with", false),
      peg$c105 = "-limit",
      peg$c106 = peg$literalExpectation("-limit", false),
      peg$c107 = function(limit) { return limit },
      peg$c108 = "",
      peg$c109 = function() { return 0 },
      peg$c110 = function(expr) { return {"kind": "Assignment", "lhs": null, "rhs": expr} },
      peg$c111 = function(first, expr) { return expr },
      peg$c112 = ":=",
      peg$c113 = peg$literalExpectation(":=", false),
      peg$c114 = function(lval, agg) {
            return {"kind": "Assignment", "lhs": lval, "rhs": agg}
          },
      peg$c115 = function(agg) {
            return {"kind": "Assignment", "lhs": null, "rhs": agg}
          },
      peg$c116 = ".",
      peg$c117 = peg$literalExpectation(".", false),
      peg$c118 = function(op, expr, where) {
            let r = {"kind": "Agg", "name": op, "expr": null, "where": where};
            if (expr) {
              r["expr"] = expr;
            }
            return r
          },
      peg$c119 = "where",
      peg$c120 = peg$literalExpectation("where", false),
      peg$c121 = function(first, rest) {
            let result = [first];
            for(let  r of rest) {
              result.push( r[3]);
            }
            return result
          },
      peg$c122 = "assert",
      peg$c123 = peg$literalExpectation("assert", false),
      peg$c124 = function(e) { return [e, text()] },
      peg$c125 = function(expr) {
            // 'assert EXPR' is equivalent to
            // 'yield EXPR ? this : error({message: "assertion failed", "expr": EXPR_text, "on": this}'
            // where EXPR_text is the literal text of EXPR.
            return {"kind": "Yield", "exprs": [{
              
            "kind": "Conditional",
              
            "cond": expr[0],
              
            "then": {"kind": "ID", "name": "this"},
              
            "else": {
                
            "kind": "Call",
                
            "name": "error",
                
            "args": [{"kind": "RecordExpr", "elems": [
                  
            {"kind": "Field", "name": "message", "value": {
                    
            "kind": "Primitive", "text": "assertion failed", "type": "string"}},
                  
            {"kind": "Field", "name": "expr", "value": {
                    
            "kind": "Primitive", "text": expr[1], "type": "string"}},
                  
            {"kind": "Field", "name": "on", "value": {
                    
            "kind": "ID", "name": "this"}}]}],
                
            "where": null}}]}
          
          },
      peg$c126 = "sort",
      peg$c127 = peg$literalExpectation("sort", false),
      peg$c128 = function(args, l) { return l },
      peg$c129 = function(args, list) {
            let argm = args;
            let op = {"kind": "Sort", "args": list, "order": "asc", "nullsfirst": false};
            if ( "r" in argm) {
              op["order"] = "desc";
            }
            if ( "nulls" in argm) {
              if (argm["nulls"] == "first") {
                op["nullsfirst"] = true;
              }
            }
            return op
          },
      peg$c130 = function(args) { return makeArgMap(args) },
      peg$c131 = "-r",
      peg$c132 = peg$literalExpectation("-r", false),
      peg$c133 = function() { return {"name": "r", "value": null} },
      peg$c134 = "-nulls",
      peg$c135 = peg$literalExpectation("-nulls", false),
      peg$c136 = "first",
      peg$c137 = peg$literalExpectation("first", false),
      peg$c138 = "last",
      peg$c139 = peg$literalExpectation("last", false),
      peg$c140 = function(where) { return {"name": "nulls", "value": where} },
      peg$c141 = "top",
      peg$c142 = peg$literalExpectation("top", false),
      peg$c143 = function(n) { return n },
      peg$c144 = "-flush",
      peg$c145 = peg$literalExpectation("-flush", false),
      peg$c146 = function(limit, flush, f) { return f },
      peg$c147 = function(limit, flush, fields) {
            let op = {"kind": "Top", "limit": 0, "args": null, "flush": false};
            if (limit) {
              op["limit"] = limit;
            }
            if (fields) {
              op["args"] = fields;
            }
            if (flush) {
              op["flush"] = true;
            }
            return op
          },
      peg$c148 = "cut",
      peg$c149 = peg$literalExpectation("cut", false),
      peg$c150 = function(args) {
            return {"kind": "Cut", "args": args}
          },
      peg$c151 = "drop",
      peg$c152 = peg$literalExpectation("drop", false),
      peg$c153 = function(args) {
            return {"kind": "Drop", "args": args}
          },
      peg$c154 = "head",
      peg$c155 = peg$literalExpectation("head", false),
      peg$c156 = function(count) { return {"kind": "Head", "count": count} },
      peg$c157 = function() { return {"kind": "Head", "count": {"kind": "Primitive", "type": "int64", "text": "1"}} },
      peg$c158 = "tail",
      peg$c159 = peg$literalExpectation("tail", false),
      peg$c160 = function(count) { return {"kind": "Tail", "count": count} },
      peg$c161 = function() { return {"kind": "Tail", "count": {"kind": "Primitive", "type": "int64", "text": "1"}} },
      peg$c162 = function(expr) {
            return {"kind": "Where", "expr": expr}
          },
      peg$c163 = "uniq",
      peg$c164 = peg$literalExpectation("uniq", false),
      peg$c165 = "-c",
      peg$c166 = peg$literalExpectation("-c", false),
      peg$c167 = function() {
            return {"kind": "Uniq", "cflag": true}
          },
      peg$c168 = function() {
            return {"kind": "Uniq", "cflag": false}
          },
      peg$c169 = "put",
      peg$c170 = peg$literalExpectation("put", false),
      peg$c171 = function(args) {
            return {"kind": "Put", "args": args}
          },
      peg$c172 = "rename",
      peg$c173 = peg$literalExpectation("rename", false),
      peg$c174 = function(first, cl) { return cl },
      peg$c175 = function(first, rest) {
            return {"kind": "Rename", "args": [first, ... rest]}
          },
      peg$c176 = "fuse",
      peg$c177 = peg$literalExpectation("fuse", false),
      peg$c178 = function() {
            return {"kind": "Fuse"}
          },
      peg$c179 = "shape",
      peg$c180 = peg$literalExpectation("shape", false),
      peg$c181 = function() {
            return {"kind": "Shape"}
          },
      peg$c182 = "join",
      peg$c183 = peg$literalExpectation("join", false),
      peg$c184 = function(style, rightInput, key, optKey, optArgs) {
            let m = {"kind": "Join", "style": style, "right_input": rightInput, "left_key": key, "right_key": key, "args": null};
            if (optKey) {
              m["right_key"] = optKey[3];
            }
            if (optArgs) {
              m["args"] = optArgs[1];
            }
            return m
          },
      peg$c185 = "anti",
      peg$c186 = peg$literalExpectation("anti", false),
      peg$c187 = function() { return "anti" },
      peg$c188 = "inner",
      peg$c189 = peg$literalExpectation("inner", false),
      peg$c190 = function() { return "inner" },
      peg$c191 = "left",
      peg$c192 = peg$literalExpectation("left", false),
      peg$c193 = function() { return "left" },
      peg$c194 = "right",
      peg$c195 = peg$literalExpectation("right", false),
      peg$c196 = function() { return "right" },
      peg$c197 = function(s) { return s },
      peg$c198 = "sample",
      peg$c199 = peg$literalExpectation("sample", false),
      peg$c200 = function(e) { return {"kind": "Sample", "expr": e} },
      peg$c201 = function(a) {
          return {"kind": "OpAssignment", "assignments": a}
        },
      peg$c202 = function(lval) { return lval },
      peg$c203 = function() { return {"kind": "ID", "name": "this"} },
      peg$c204 = "load",
      peg$c205 = peg$literalExpectation("load", false),
      peg$c206 = function(pool, branch, author, message, meta) {
            return {"kind": "Load", "pool": pool, "branch": branch, "author": author, "message": message, "meta": meta}
          },
      peg$c207 = "author",
      peg$c208 = peg$literalExpectation("author", false),
      peg$c209 = function(val) { return val },
      peg$c210 = "message",
      peg$c211 = peg$literalExpectation("message", false),
      peg$c212 = "meta",
      peg$c213 = peg$literalExpectation("meta", false),
      peg$c214 = "@",
      peg$c215 = peg$literalExpectation("@", false),
      peg$c216 = function(branch) { return branch },
      peg$c217 = function(source) {
            return {"kind": "From", "trunks": [{"kind": "Trunk", "source": source}]}
          },
      peg$c218 = "file",
      peg$c219 = peg$literalExpectation("file", false),
      peg$c220 = function(path, format, sort_key) {
            return {"kind": "File", "path": path, "format": format, "sort_key": sort_key}
          },
      peg$c221 = function(body) { return body },
      peg$c222 = "pool",
      peg$c223 = peg$literalExpectation("pool", false),
      peg$c224 = function(spec, at) {
            return {"kind": "Pool", "spec": spec, "at": at}
          },
      peg$c225 = "get",
      peg$c226 = peg$literalExpectation("get", false),
      peg$c227 = function(url, format, sort_key) {
            return {"kind": "HTTP", "url": url, "format": format, "sort_key": sort_key}
          },
      peg$c228 = "http:",
      peg$c229 = peg$literalExpectation("http:", false),
      peg$c230 = "https:",
      peg$c231 = peg$literalExpectation("https:", false),
      peg$c232 = /^[0-9a-zA-Z!@$%\^&*_=<>,.\/?:[\]{}~+\-]/,
      peg$c233 = peg$classExpectation([["0", "9"], ["a", "z"], ["A", "Z"], "!", "@", "$", "%", "^", "&", "*", "_", "=", "<", ">", ",", ".", "/", "?", ":", "[", "]", "{", "}", "~", "+", "-"], false, false),
      peg$c234 = "at",
      peg$c235 = peg$literalExpectation("at", false),
      peg$c236 = function(id) { return id },
      peg$c237 = /^[0-9a-zA-Z]/,
      peg$c238 = peg$classExpectation([["0", "9"], ["a", "z"], ["A", "Z"]], false, false),
      peg$c239 = function(pool, commit, meta, tap) {
            return {"pool": pool, "commit": commit, "meta": meta, "tap": tap}
          },
      peg$c240 = function(meta) {
            return {"pool": null, "commit": null, "meta": meta}
          },
      peg$c241 = function(commit) { return commit },
      peg$c242 = function(meta) { return meta },
      peg$c243 = function() { return {"kind": "Glob", "pattern": "*"} },
      peg$c244 = function(name) { return {"kind": "String", "text": name} },
      peg$c245 = "order",
      peg$c246 = peg$literalExpectation("order", false),
      peg$c247 = function(keys, order) {
            return {"kind": "SortKey", "keys": keys, "order": order}
          },
      peg$c248 = "tap",
      peg$c249 = peg$literalExpectation("tap", false),
      peg$c250 = function() { return true },
      peg$c251 = function() { return false },
      peg$c252 = "format",
      peg$c253 = peg$literalExpectation("format", false),
      peg$c254 = ":asc",
      peg$c255 = peg$literalExpectation(":asc", false),
      peg$c256 = function() { return "asc" },
      peg$c257 = ":desc",
      peg$c258 = peg$literalExpectation(":desc", false),
      peg$c259 = function() { return "desc" },
      peg$c260 = "pass",
      peg$c261 = peg$literalExpectation("pass", false),
      peg$c262 = function() {
            return {"kind": "Pass"}
          },
      peg$c263 = "explode",
      peg$c264 = peg$literalExpectation("explode", false),
      peg$c265 = function(args, typ, as) {
            return {"kind": "Explode", "args": args, "as": as, "type": typ}
          },
      peg$c266 = "merge",
      peg$c267 = peg$literalExpectation("merge", false),
      peg$c268 = function(expr) {
            return {"kind": "Merge", "expr": expr}
          },
      peg$c269 = "over",
      peg$c270 = peg$literalExpectation("over", false),
      peg$c271 = function(exprs, locals, body) {
            return {"kind": "Over", "exprs": exprs, "locals": locals, "body": body}
          },
      peg$c272 = function(first, a) { return a },
      peg$c273 = function(name, opt) {
            let m = {"name": name, "expr": {"kind": "ID", "name": name}};
            if (opt) {
               m["expr"] = opt[3];
            }
            return m
          },
      peg$c274 = "yield",
      peg$c275 = peg$literalExpectation("yield", false),
      peg$c276 = function(exprs) {
            return {"kind": "Yield", "exprs": exprs}
          },
      peg$c277 = function(typ) { return typ },
      peg$c278 = function(lhs) { return lhs },
      peg$c280 = function(first, rest) {
            let result = [first];

            for(let  r of rest) {
              result.push( r[3]);
            }

            return result
          },
      peg$c281 = function(first, rest) {
          return [first, ... rest]
        },
      peg$c282 = function(lhs, rhs) { return {"kind": "Assignment", "lhs": lhs, "rhs": rhs} },
      peg$c283 = "?",
      peg$c284 = peg$literalExpectation("?", false),
      peg$c285 = function(cond, opt) {
            if (opt) {
              let Then = opt[3];
              let Else = opt[7];
              return {"kind": "Conditional", "cond": cond, "then": Then, "else": Else}
            }
            return cond
          },
      peg$c286 = function(first, op, expr) { return [op, expr] },
      peg$c287 = function(first, rest) {
              return makeBinaryExprChain(first, rest)
          },
      peg$c288 = function(lhs) { return text() },
      peg$c289 = function(lhs, opAndRHS) {
            if (!opAndRHS) {
              return lhs
            }
            let op = opAndRHS[1];
            let rhs = opAndRHS[3];
            return {"kind": "BinaryExpr", "op": op, "lhs": lhs, "rhs": rhs}
          },
      peg$c290 = "+",
      peg$c291 = peg$literalExpectation("+", false),
      peg$c292 = "-",
      peg$c293 = peg$literalExpectation("-", false),
      peg$c294 = "/",
      peg$c295 = peg$literalExpectation("/", false),
      peg$c296 = "%",
      peg$c297 = peg$literalExpectation("%", false),
      peg$c298 = function(e) {
              return {"kind": "UnaryExpr", "op": "!", "operand": e}
          },
      peg$c299 = function(e) {
              return {"kind": "UnaryExpr", "op": "-", "operand": e}
          },
      peg$c300 = "not",
      peg$c301 = peg$literalExpectation("not", false),
      peg$c302 = "select",
      peg$c303 = peg$literalExpectation("select", false),
      peg$c304 = function(typ, expr) {
            return {"kind": "Cast", "expr": expr, "type": typ}
          },
      peg$c305 = "regexp",
      peg$c306 = peg$literalExpectation("regexp", false),
      peg$c307 = function(arg0Text, arg1, where) {
            let arg0 = {"kind": "Primitive", "type": "string", "text": arg0Text};
            return {"kind": "Call", "name": "regexp", "args": [arg0, arg1], "where": where}
          },
      peg$c308 = "regexp_replace",
      peg$c309 = peg$literalExpectation("regexp_replace", false),
      peg$c310 = function(arg0, arg1Text, arg2, where) {
            let arg1 = {"kind": "Primitive", "type": "string", "text": arg1Text};
            return {"kind": "Call", "name": "regexp_replace", "args": [arg0, arg1, arg2], "where": where}
          },
      peg$c311 = function(fn, args, where) {
            return {"kind": "Call", "name": fn, "args": args, "where": where}
          },
      peg$c312 = function(o) { return [o] },
      peg$c313 = "grep",
      peg$c314 = peg$literalExpectation("grep", false),
      peg$c315 = function(pattern, opt) {
            let m = {"kind": "Grep", "pattern": pattern, "expr": {"kind": "ID", "name": "this"}};
            if (opt) {
              m["expr"] = opt[2];
            }
            return m
          },
      peg$c316 = function(s) {
            return {"kind": "String", "text": s}
          },
      peg$c317 = function() { return [] },
      peg$c318 = function(first, e) { return e },
      peg$c319 = "]",
      peg$c320 = peg$literalExpectation("]", false),
      peg$c321 = function(from, to) {
            return ["[", {"kind": "BinaryExpr", "op": ":",
                                  
            "lhs": from, "rhs": to}]
          
          },
      peg$c322 = function(to) {
            return ["[", {"kind": "BinaryExpr", "op": ":",
                                  
            "lhs": null, "rhs": to}]
          
          },
      peg$c323 = function(expr) { return ["[", expr] },
      peg$c324 = function(id) { return [".", id] },
      peg$c325 = function(exprs, locals, body) {
            return {"kind": "OverExpr", "locals": locals, "exprs": exprs, "body": body}
          },
      peg$c326 = "}",
      peg$c327 = peg$literalExpectation("}", false),
      peg$c328 = function(elems) {
            return {"kind": "RecordExpr", "elems": elems}
          },
      peg$c329 = function(elem) { return elem },
      peg$c330 = function(expr) {
            return {"kind": "Spread", "expr": expr}
          },
      peg$c331 = function(name, value) {
            return {"kind": "Field", "name": name, "value": value}
          },
      peg$c332 = function(elems) {
            return {"kind": "ArrayExpr", "elems": elems}
          },
      peg$c333 = "|[",
      peg$c334 = peg$literalExpectation("|[", false),
      peg$c335 = "]|",
      peg$c336 = peg$literalExpectation("]|", false),
      peg$c337 = function(elems) {
            return {"kind": "SetExpr", "elems": elems}
          },
      peg$c338 = function(e) { return {"kind": "VectorValue", "expr": e} },
      peg$c339 = "|{",
      peg$c340 = peg$literalExpectation("|{", false),
      peg$c341 = "}|",
      peg$c342 = peg$literalExpectation("}|", false),
      peg$c343 = function(exprs) {
            return {"kind": "MapExpr", "entries": exprs}
          },
      peg$c344 = function(e) { return e },
      peg$c345 = function(key, value) {
            return {"key": key, "value": value}
          },
      peg$c346 = function(selection, from, joins, where, groupby, having, orderby, limit) {
            return {
              
            "kind": "SQLExpr",
              
            "select": selection,
              
            "from": from,
              
            "joins": joins,
              
            "where": where,
              
            "group_by": groupby,
              
            "having": having,
              
            "order_by": orderby,
              
            "limit": limit}
          
          },
      peg$c347 = function(assignments) { return assignments },
      peg$c348 = function(rhs, opt) {
            let m = {"kind": "Assignment", "lhs": null, "rhs": rhs};
            if (opt) {
              m["lhs"] = opt[3];
            }
            return m
          },
      peg$c349 = function(table, alias) {
            return {"table": table, "alias": alias}
          },
      peg$c350 = function(first, join) { return join },
      peg$c351 = function(style, table, alias, leftKey, rightKey) {
            return {
              
            "table": table,
              
            "style": style,
              
            "left_key": leftKey,
              
            "right_key": rightKey,
              
            "alias": alias}
          
          },
      peg$c352 = function(style) { return style },
      peg$c353 = function(keys, order) {
            return {"kind": "SQLOrderBy", "keys": keys, "order": order}
          },
      peg$c354 = function(dir) { return dir },
      peg$c355 = function(count) { return count },
      peg$c356 = peg$literalExpectation("select", true),
      peg$c357 = function() { return "select" },
      peg$c358 = "as",
      peg$c359 = peg$literalExpectation("as", true),
      peg$c360 = function() { return "as" },
      peg$c361 = peg$literalExpectation("from", true),
      peg$c362 = function() { return "from" },
      peg$c363 = peg$literalExpectation("join", true),
      peg$c364 = function() { return "join" },
      peg$c365 = peg$literalExpectation("where", true),
      peg$c366 = function() { return "where" },
      peg$c367 = "group",
      peg$c368 = peg$literalExpectation("group", true),
      peg$c369 = function() { return "group" },
      peg$c370 = "by",
      peg$c371 = peg$literalExpectation("by", true),
      peg$c372 = function() { return "by" },
      peg$c373 = "having",
      peg$c374 = peg$literalExpectation("having", true),
      peg$c375 = function() { return "having" },
      peg$c376 = peg$literalExpectation("order", true),
      peg$c377 = function() { return "order" },
      peg$c378 = "on",
      peg$c379 = peg$literalExpectation("on", true),
      peg$c380 = function() { return "on" },
      peg$c381 = "limit",
      peg$c382 = peg$literalExpectation("limit", true),
      peg$c383 = function() { return "limit" },
      peg$c384 = "asc",
      peg$c385 = peg$literalExpectation("asc", true),
      peg$c386 = "desc",
      peg$c387 = peg$literalExpectation("desc", true),
      peg$c388 = peg$literalExpectation("anti", true),
      peg$c389 = peg$literalExpectation("left", true),
      peg$c390 = peg$literalExpectation("right", true),
      peg$c391 = peg$literalExpectation("inner", true),
      peg$c392 = function(v) {
            return {"kind": "Primitive", "type": "net", "text": v}
          },
      peg$c393 = function(v) {
            return {"kind": "Primitive", "type": "ip", "text": v}
          },
      peg$c394 = function(v) {
            return {"kind": "Primitive", "type": "float64", "text": v}
          },
      peg$c395 = function(v) {
            return {"kind": "Primitive", "type": "int64", "text": v}
          },
      peg$c396 = "true",
      peg$c397 = peg$literalExpectation("true", false),
      peg$c398 = function() { return {"kind": "Primitive", "type": "bool", "text": "true"} },
      peg$c399 = "false",
      peg$c400 = peg$literalExpectation("false", false),
      peg$c401 = function() { return {"kind": "Primitive", "type": "bool", "text": "false"} },
      peg$c402 = "null",
      peg$c403 = peg$literalExpectation("null", false),
      peg$c404 = function() { return {"kind": "Primitive", "type": "null", "text": ""} },
      peg$c405 = "0x",
      peg$c406 = peg$literalExpectation("0x", false),
      peg$c407 = function() {
      	return {"kind": "Primitive", "type": "bytes", "text": text()}
        },
      peg$c408 = function(typ) {
            return {"kind": "TypeValue", "value": typ}
          },
      peg$c409 = function(name) { return name },
      peg$c410 = function(name, opt) {
            if (opt) {
              return {"kind": "TypeDef", "name": name, "type": opt[3]}
            }
            return {"kind": "TypeName", "name": name}
          },
      peg$c411 = function(name) {
            return {"kind": "TypeName", "name": name}
          },
      peg$c412 = function(u) { return u },
      peg$c413 = function(types) {
            return {"kind": "TypeUnion", "types": types}
          },
      peg$c414 = function(fields) {
            return {"kind": "TypeRecord", "fields": fields}
          },
      peg$c415 = function(typ) {
            return {"kind": "TypeArray", "type": typ}
          },
      peg$c416 = function(typ) {
            return {"kind": "TypeSet", "type": typ}
          },
      peg$c417 = function(keyType, valType) {
            return {"kind": "TypeMap", "key_type": keyType, "val_type": valType}
          },
      peg$c418 = function(v) {
            if (v.length == 0) {
              return {"kind": "Primitive", "type": "string", "text": ""}
            }
            return makeTemplateExprChain(v)
          },
      peg$c419 = "\"",
      peg$c420 = peg$literalExpectation("\"", false),
      peg$c421 = "'",
      peg$c422 = peg$literalExpectation("'", false),
      peg$c423 = function(v) {
            return {"kind": "Primitive", "type": "string", "text": joinChars(v)}
          },
      peg$c424 = "\\",
      peg$c425 = peg$literalExpectation("\\", false),
      peg$c426 = "${",
      peg$c427 = peg$literalExpectation("${", false),
      peg$c428 = function(e) {
            return {
              
            "kind": "Cast",
              
            "expr": e,
              
            "type": {
                
            "kind": "TypeValue",
                
            "value": {"kind": "TypePrimitive", "name": "string"}}}
          
          },
      peg$c429 = "uint8",
      peg$c430 = peg$literalExpectation("uint8", false),
      peg$c431 = "uint16",
      peg$c432 = peg$literalExpectation("uint16", false),
      peg$c433 = "uint32",
      peg$c434 = peg$literalExpectation("uint32", false),
      peg$c435 = "uint64",
      peg$c436 = peg$literalExpectation("uint64", false),
      peg$c437 = "int8",
      peg$c438 = peg$literalExpectation("int8", false),
      peg$c439 = "int16",
      peg$c440 = peg$literalExpectation("int16", false),
      peg$c441 = "int32",
      peg$c442 = peg$literalExpectation("int32", false),
      peg$c443 = "int64",
      peg$c444 = peg$literalExpectation("int64", false),
      peg$c445 = "float16",
      peg$c446 = peg$literalExpectation("float16", false),
      peg$c447 = "float32",
      peg$c448 = peg$literalExpectation("float32", false),
      peg$c449 = "float64",
      peg$c450 = peg$literalExpectation("float64", false),
      peg$c451 = "bool",
      peg$c452 = peg$literalExpectation("bool", false),
      peg$c453 = "string",
      peg$c454 = peg$literalExpectation("string", false),
      peg$c455 = "duration",
      peg$c456 = peg$literalExpectation("duration", false),
      peg$c457 = "time",
      peg$c458 = peg$literalExpectation("time", false),
      peg$c459 = "bytes",
      peg$c460 = peg$literalExpectation("bytes", false),
      peg$c461 = "ip",
      peg$c462 = peg$literalExpectation("ip", false),
      peg$c463 = "net",
      peg$c464 = peg$literalExpectation("net", false),
      peg$c465 = function() {
                return {"kind": "TypePrimitive", "name": text()}
              },
      peg$c466 = function(name, typ) {
            return {"name": name, "type": typ}
          },
      peg$c467 = "and",
      peg$c468 = peg$literalExpectation("and", false),
      peg$c469 = "AND",
      peg$c470 = peg$literalExpectation("AND", false),
      peg$c471 = function() { return "and" },
      peg$c472 = "or",
      peg$c473 = peg$literalExpectation("or", false),
      peg$c474 = "OR",
      peg$c475 = peg$literalExpectation("OR", false),
      peg$c476 = function() { return "or" },
      peg$c478 = "NOT",
      peg$c479 = peg$literalExpectation("NOT", false),
      peg$c480 = function() { return "not" },
      peg$c481 = peg$literalExpectation("by", false),
      peg$c482 = /^[A-Za-z_$]/,
      peg$c483 = peg$classExpectation([["A", "Z"], ["a", "z"], "_", "$"], false, false),
      peg$c484 = /^[0-9]/,
      peg$c485 = peg$classExpectation([["0", "9"]], false, false),
      peg$c486 = function(id) { return {"kind": "ID", "name": id} },
      peg$c487 = "$",
      peg$c488 = peg$literalExpectation("$", false),
      peg$c489 = function(first, id) { return id },
      peg$c490 = "T",
      peg$c491 = peg$literalExpectation("T", false),
      peg$c492 = function() {
            return {"kind": "Primitive", "type": "time", "text": text()}
          },
      peg$c493 = "Z",
      peg$c494 = peg$literalExpectation("Z", false),
      peg$c495 = function() {
            return {"kind": "Primitive", "type": "duration", "text": text()}
          },
      peg$c496 = "ns",
      peg$c497 = peg$literalExpectation("ns", false),
      peg$c498 = "us",
      peg$c499 = peg$literalExpectation("us", false),
      peg$c500 = "ms",
      peg$c501 = peg$literalExpectation("ms", false),
      peg$c502 = "s",
      peg$c503 = peg$literalExpectation("s", false),
      peg$c504 = "m",
      peg$c505 = peg$literalExpectation("m", false),
      peg$c506 = "h",
      peg$c507 = peg$literalExpectation("h", false),
      peg$c508 = "d",
      peg$c509 = peg$literalExpectation("d", false),
      peg$c510 = "w",
      peg$c511 = peg$literalExpectation("w", false),
      peg$c512 = "y",
      peg$c513 = peg$literalExpectation("y", false),
      peg$c514 = function(a, b) {
            return joinChars(a) + b
          },
      peg$c515 = "::",
      peg$c516 = peg$literalExpectation("::", false),
      peg$c517 = function(a, b, d, e) {
            return a + joinChars(b) + "::" + joinChars(d) + e
          },
      peg$c518 = function(a, b) {
            return "::" + joinChars(a) + b
          },
      peg$c519 = function(a, b) {
            return a + joinChars(b) + "::"
          },
      peg$c520 = function() {
            return "::"
          },
      peg$c521 = function(v) { return ":" + v },
      peg$c522 = function(v) { return v + ":" },
      peg$c523 = function(a, m) {
            return a + "/" + m.toString();
          },
      peg$c524 = function(a, m) {
            return a + "/" + m;
          },
      peg$c525 = function(s) { return parseInt(s) },
      peg$c526 = function() {
            return text()
          },
      peg$c527 = "e",
      peg$c528 = peg$literalExpectation("e", true),
      peg$c529 = /^[+\-]/,
      peg$c530 = peg$classExpectation(["+", "-"], false, false),
      peg$c531 = "NaN",
      peg$c532 = peg$literalExpectation("NaN", false),
      peg$c533 = "Inf",
      peg$c534 = peg$literalExpectation("Inf", false),
      peg$c535 = /^[0-9a-fA-F]/,
      peg$c536 = peg$classExpectation([["0", "9"], ["a", "f"], ["A", "F"]], false, false),
      peg$c537 = function(v) { return joinChars(v) },
      peg$c538 = peg$anyExpectation(),
      peg$c539 = function(head, tail) { return head + joinChars(tail) },
      peg$c540 = /^[a-zA-Z_.:\/%#@~]/,
      peg$c541 = peg$classExpectation([["a", "z"], ["A", "Z"], "_", ".", ":", "/", "%", "#", "@", "~"], false, false),
      peg$c542 = function(head, tail) {
            return head + joinChars(tail)
          },
      peg$c543 = function() { return "*" },
      peg$c544 = function() { return "=" },
      peg$c545 = function() { return "\\*" },
      peg$c546 = "b",
      peg$c547 = peg$literalExpectation("b", false),
      peg$c548 = function() { return "\b" },
      peg$c549 = "f",
      peg$c550 = peg$literalExpectation("f", false),
      peg$c551 = function() { return "\f" },
      peg$c552 = "n",
      peg$c553 = peg$literalExpectation("n", false),
      peg$c554 = function() { return "\n" },
      peg$c555 = "r",
      peg$c556 = peg$literalExpectation("r", false),
      peg$c557 = function() { return "\r" },
      peg$c558 = "t",
      peg$c559 = peg$literalExpectation("t", false),
      peg$c560 = function() { return "\t" },
      peg$c561 = "v",
      peg$c562 = peg$literalExpectation("v", false),
      peg$c563 = function() { return "\v" },
      peg$c564 = "u",
      peg$c565 = peg$literalExpectation("u", false),
      peg$c566 = function(chars) {
            return makeUnicodeChar(chars)
          },
      peg$c567 = /^[^\/\\]/,
      peg$c568 = peg$classExpectation(["/", "\\"], true, false),
      peg$c569 = /^[\0-\x1F\\]/,
      peg$c570 = peg$classExpectation([["\0", "\x1F"], "\\"], false, false),
      peg$c571 = peg$otherExpectation("whitespace"),
      peg$c572 = "\t",
      peg$c573 = peg$literalExpectation("\t", false),
      peg$c574 = "\x0B",
      peg$c575 = peg$literalExpectation("\x0B", false),
      peg$c576 = "\f",
      peg$c577 = peg$literalExpectation("\f", false),
      peg$c578 = " ",
      peg$c579 = peg$literalExpectation(" ", false),
      peg$c580 = "\xA0",
      peg$c581 = peg$literalExpectation("\xA0", false),
      peg$c582 = "\uFEFF",
      peg$c583 = peg$literalExpectation("\uFEFF", false),
      peg$c584 = /^[\n\r\u2028\u2029]/,
      peg$c585 = peg$classExpectation(["\n", "\r", "\u2028", "\u2029"], false, false),
      peg$c586 = peg$otherExpectation("comment"),
      peg$c591 = "//",
      peg$c592 = peg$literalExpectation("//", false),

      peg$currPos          = 0,
      peg$savedPos         = 0,
      peg$posDetailsCache  = [{ line: 1, column: 1 }],
      peg$maxFailPos       = 0,
      peg$maxFailExpected  = [],
      peg$silentFails      = 0,

      peg$result;

  if ("startRule" in options) {
    if (!(options.startRule in peg$startRuleFunctions)) {
      throw new Error("Can't start parsing from rule \"" + options.startRule + "\".");
    }

    peg$startRuleFunction = peg$startRuleFunctions[options.startRule];
  }

  function text() {
    return input.substring(peg$savedPos, peg$currPos);
  }

  function peg$literalExpectation(text, ignoreCase) {
    return { type: "literal", text: text, ignoreCase: ignoreCase };
  }

  function peg$classExpectation(parts, inverted, ignoreCase) {
    return { type: "class", parts: parts, inverted: inverted, ignoreCase: ignoreCase };
  }

  function peg$anyExpectation() {
    return { type: "any" };
  }

  function peg$endExpectation() {
    return { type: "end" };
  }

  function peg$otherExpectation(description) {
    return { type: "other", description: description };
  }

  function peg$computePosDetails(pos) {
    var details = peg$posDetailsCache[pos], p;

    if (details) {
      return details;
    } else {
      p = pos - 1;
      while (!peg$posDetailsCache[p]) {
        p--;
      }

      details = peg$posDetailsCache[p];
      details = {
        line:   details.line,
        column: details.column
      };

      while (p < pos) {
        if (input.charCodeAt(p) === 10) {
          details.line++;
          details.column = 1;
        } else {
          details.column++;
        }

        p++;
      }

      peg$posDetailsCache[pos] = details;
      return details;
    }
  }

  function peg$computeLocation(startPos, endPos) {
    var startPosDetails = peg$computePosDetails(startPos),
        endPosDetails   = peg$computePosDetails(endPos);

    return {
      start: {
        offset: startPos,
        line:   startPosDetails.line,
        column: startPosDetails.column
      },
      end: {
        offset: endPos,
        line:   endPosDetails.line,
        column: endPosDetails.column
      }
    };
  }

  function peg$fail(expected) {
    if (peg$currPos < peg$maxFailPos) { return; }

    if (peg$currPos > peg$maxFailPos) {
      peg$maxFailPos = peg$currPos;
      peg$maxFailExpected = [];
    }

    peg$maxFailExpected.push(expected);
  }

  function peg$buildStructuredError(expected, found, location) {
    return new peg$SyntaxError(
      peg$SyntaxError.buildMessage(expected, found),
      expected,
      found,
      location
    );
  }

  function peg$parsestart() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseScope();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseEOF();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c0(s2);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parse__();
      if (s1 !== peg$FAILED) {
        s2 = peg$parseSeq();
        if (s2 !== peg$FAILED) {
          s3 = peg$parse__();
          if (s3 !== peg$FAILED) {
            s4 = peg$parseEOF();
            if (s4 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c1(s2);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseScope() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = [];
    s2 = peg$parseDecl();
    if (s2 !== peg$FAILED) {
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parseDecl();
      }
    } else {
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseSeq();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c2(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSeq() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseOperation();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseSeqTail();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseSeqTail();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c3(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSeqTail() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      s2 = peg$parsePipe();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseOperation();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c4(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseDecl() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parseConstDecl();
    if (s1 === peg$FAILED) {
      s1 = peg$parseFuncDecl();
      if (s1 === peg$FAILED) {
        s1 = peg$parseOpDecl();
      }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c5(s1);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseConstDecl() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5) === peg$c6) {
      s1 = peg$c6;
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c7); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseIdentifierName();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 61) {
              s5 = peg$c8;
              peg$currPos++;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c9); }
            }
            if (s5 !== peg$FAILED) {
              s6 = peg$parse__();
              if (s6 !== peg$FAILED) {
                s7 = peg$parseConditionalExpr();
                if (s7 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c10(s3, s7);
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 4) === peg$c11) {
        s1 = peg$c11;
        peg$currPos += 4;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c12); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parse_();
        if (s2 !== peg$FAILED) {
          s3 = peg$parseIdentifierName();
          if (s3 === peg$FAILED) {
            s3 = peg$parseQuotedString();
          }
          if (s3 !== peg$FAILED) {
            s4 = peg$parse__();
            if (s4 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 61) {
                s5 = peg$c8;
                peg$currPos++;
              } else {
                s5 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c9); }
              }
              if (s5 !== peg$FAILED) {
                s6 = peg$parse__();
                if (s6 !== peg$FAILED) {
                  s7 = peg$parseType();
                  if (s7 !== peg$FAILED) {
                    peg$savedPos = s0;
                    s1 = peg$c13(s3, s7);
                    s0 = s1;
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseFuncDecl() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10, s11, s12, s13, s14, s15, s16, s17;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c14) {
      s1 = peg$c14;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c15); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseIdentifierName();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 40) {
              s5 = peg$c16;
              peg$currPos++;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c17); }
            }
            if (s5 !== peg$FAILED) {
              s6 = peg$parse__();
              if (s6 !== peg$FAILED) {
                s7 = peg$parseIdentifierNames();
                if (s7 !== peg$FAILED) {
                  s8 = peg$parse__();
                  if (s8 !== peg$FAILED) {
                    if (input.charCodeAt(peg$currPos) === 41) {
                      s9 = peg$c18;
                      peg$currPos++;
                    } else {
                      s9 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c19); }
                    }
                    if (s9 !== peg$FAILED) {
                      s10 = peg$parse__();
                      if (s10 !== peg$FAILED) {
                        if (input.charCodeAt(peg$currPos) === 58) {
                          s11 = peg$c20;
                          peg$currPos++;
                        } else {
                          s11 = peg$FAILED;
                          if (peg$silentFails === 0) { peg$fail(peg$c21); }
                        }
                        if (s11 !== peg$FAILED) {
                          s12 = peg$parse__();
                          if (s12 !== peg$FAILED) {
                            if (input.charCodeAt(peg$currPos) === 40) {
                              s13 = peg$c16;
                              peg$currPos++;
                            } else {
                              s13 = peg$FAILED;
                              if (peg$silentFails === 0) { peg$fail(peg$c17); }
                            }
                            if (s13 !== peg$FAILED) {
                              s14 = peg$parse__();
                              if (s14 !== peg$FAILED) {
                                s15 = peg$parseConditionalExpr();
                                if (s15 !== peg$FAILED) {
                                  s16 = peg$parse__();
                                  if (s16 !== peg$FAILED) {
                                    if (input.charCodeAt(peg$currPos) === 41) {
                                      s17 = peg$c18;
                                      peg$currPos++;
                                    } else {
                                      s17 = peg$FAILED;
                                      if (peg$silentFails === 0) { peg$fail(peg$c19); }
                                    }
                                    if (s17 !== peg$FAILED) {
                                      peg$savedPos = s0;
                                      s1 = peg$c22(s3, s7, s15);
                                      s0 = s1;
                                    } else {
                                      peg$currPos = s0;
                                      s0 = peg$FAILED;
                                    }
                                  } else {
                                    peg$currPos = s0;
                                    s0 = peg$FAILED;
                                  }
                                } else {
                                  peg$currPos = s0;
                                  s0 = peg$FAILED;
                                }
                              } else {
                                peg$currPos = s0;
                                s0 = peg$FAILED;
                              }
                            } else {
                              peg$currPos = s0;
                              s0 = peg$FAILED;
                            }
                          } else {
                            peg$currPos = s0;
                            s0 = peg$FAILED;
                          }
                        } else {
                          peg$currPos = s0;
                          s0 = peg$FAILED;
                        }
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseOpDecl() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10, s11, s12, s13, s14, s15, s16, s17;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c23) {
      s1 = peg$c23;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c24); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseIdentifierName();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 40) {
              s5 = peg$c16;
              peg$currPos++;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c17); }
            }
            if (s5 !== peg$FAILED) {
              s6 = peg$parse__();
              if (s6 !== peg$FAILED) {
                s7 = peg$parseParams();
                if (s7 !== peg$FAILED) {
                  s8 = peg$parse__();
                  if (s8 !== peg$FAILED) {
                    if (input.charCodeAt(peg$currPos) === 41) {
                      s9 = peg$c18;
                      peg$currPos++;
                    } else {
                      s9 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c19); }
                    }
                    if (s9 !== peg$FAILED) {
                      s10 = peg$parse__();
                      if (s10 !== peg$FAILED) {
                        if (input.charCodeAt(peg$currPos) === 58) {
                          s11 = peg$c20;
                          peg$currPos++;
                        } else {
                          s11 = peg$FAILED;
                          if (peg$silentFails === 0) { peg$fail(peg$c21); }
                        }
                        if (s11 !== peg$FAILED) {
                          s12 = peg$parse__();
                          if (s12 !== peg$FAILED) {
                            if (input.charCodeAt(peg$currPos) === 40) {
                              s13 = peg$c16;
                              peg$currPos++;
                            } else {
                              s13 = peg$FAILED;
                              if (peg$silentFails === 0) { peg$fail(peg$c17); }
                            }
                            if (s13 !== peg$FAILED) {
                              s14 = peg$parse__();
                              if (s14 !== peg$FAILED) {
                                s15 = peg$parseSeq();
                                if (s15 !== peg$FAILED) {
                                  s16 = peg$parse__();
                                  if (s16 !== peg$FAILED) {
                                    if (input.charCodeAt(peg$currPos) === 41) {
                                      s17 = peg$c18;
                                      peg$currPos++;
                                    } else {
                                      s17 = peg$FAILED;
                                      if (peg$silentFails === 0) { peg$fail(peg$c19); }
                                    }
                                    if (s17 !== peg$FAILED) {
                                      peg$savedPos = s0;
                                      s1 = peg$c25(s3, s7, s15);
                                      s0 = s1;
                                    } else {
                                      peg$currPos = s0;
                                      s0 = peg$FAILED;
                                    }
                                  } else {
                                    peg$currPos = s0;
                                    s0 = peg$FAILED;
                                  }
                                } else {
                                  peg$currPos = s0;
                                  s0 = peg$FAILED;
                                }
                              } else {
                                peg$currPos = s0;
                                s0 = peg$FAILED;
                              }
                            } else {
                              peg$currPos = s0;
                              s0 = peg$FAILED;
                            }
                          } else {
                            peg$currPos = s0;
                            s0 = peg$FAILED;
                          }
                        } else {
                          peg$currPos = s0;
                          s0 = peg$FAILED;
                        }
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseParams() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseParam();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s5 = peg$c26;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c27); }
        }
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseParam();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c28(s1, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 44) {
            s5 = peg$c26;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c27); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseParam();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c28(s1, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c3(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseParam() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 3) === peg$c29) {
      s1 = peg$c29;
      peg$currPos += 3;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c30); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c31();
    }
    s0 = s1;
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseIdentifierName();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c32(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseOperation() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c33) {
      s1 = peg$c33;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c34); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 40) {
          s3 = peg$c16;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c17); }
        }
        if (s3 !== peg$FAILED) {
          s4 = [];
          s5 = peg$parseLeg();
          if (s5 !== peg$FAILED) {
            while (s5 !== peg$FAILED) {
              s4.push(s5);
              s5 = peg$parseLeg();
            }
          } else {
            s4 = peg$FAILED;
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parse__();
            if (s5 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 41) {
                s6 = peg$c18;
                peg$currPos++;
              } else {
                s6 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c19); }
              }
              if (s6 !== peg$FAILED) {
                peg$savedPos = s0;
                s1 = peg$c35(s4);
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 6) === peg$c36) {
        s1 = peg$c36;
        peg$currPos += 6;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c37); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parse_();
        if (s2 !== peg$FAILED) {
          s3 = peg$parseConditionalExpr();
          if (s3 !== peg$FAILED) {
            s4 = peg$parse_();
            if (s4 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 40) {
                s5 = peg$c16;
                peg$currPos++;
              } else {
                s5 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c17); }
              }
              if (s5 !== peg$FAILED) {
                s6 = [];
                s7 = peg$parseSwitchLeg();
                if (s7 !== peg$FAILED) {
                  while (s7 !== peg$FAILED) {
                    s6.push(s7);
                    s7 = peg$parseSwitchLeg();
                  }
                } else {
                  s6 = peg$FAILED;
                }
                if (s6 !== peg$FAILED) {
                  s7 = peg$parse__();
                  if (s7 !== peg$FAILED) {
                    if (input.charCodeAt(peg$currPos) === 41) {
                      s8 = peg$c18;
                      peg$currPos++;
                    } else {
                      s8 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c19); }
                    }
                    if (s8 !== peg$FAILED) {
                      peg$savedPos = s0;
                      s1 = peg$c38(s3, s6);
                      s0 = s1;
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        if (input.substr(peg$currPos, 6) === peg$c36) {
          s1 = peg$c36;
          peg$currPos += 6;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c37); }
        }
        if (s1 !== peg$FAILED) {
          s2 = peg$parse__();
          if (s2 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 40) {
              s3 = peg$c16;
              peg$currPos++;
            } else {
              s3 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c17); }
            }
            if (s3 !== peg$FAILED) {
              s4 = [];
              s5 = peg$parseSwitchLeg();
              if (s5 !== peg$FAILED) {
                while (s5 !== peg$FAILED) {
                  s4.push(s5);
                  s5 = peg$parseSwitchLeg();
                }
              } else {
                s4 = peg$FAILED;
              }
              if (s4 !== peg$FAILED) {
                s5 = peg$parse__();
                if (s5 !== peg$FAILED) {
                  if (input.charCodeAt(peg$currPos) === 41) {
                    s6 = peg$c18;
                    peg$currPos++;
                  } else {
                    s6 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c19); }
                  }
                  if (s6 !== peg$FAILED) {
                    peg$savedPos = s0;
                    s1 = peg$c39(s4);
                    s0 = s1;
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          if (input.substr(peg$currPos, 4) === peg$c40) {
            s1 = peg$c40;
            peg$currPos += 4;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c41); }
          }
          if (s1 !== peg$FAILED) {
            s2 = peg$parse__();
            if (s2 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 40) {
                s3 = peg$c16;
                peg$currPos++;
              } else {
                s3 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c17); }
              }
              if (s3 !== peg$FAILED) {
                s4 = [];
                s5 = peg$parseFromLeg();
                if (s5 !== peg$FAILED) {
                  while (s5 !== peg$FAILED) {
                    s4.push(s5);
                    s5 = peg$parseFromLeg();
                  }
                } else {
                  s4 = peg$FAILED;
                }
                if (s4 !== peg$FAILED) {
                  s5 = peg$parse__();
                  if (s5 !== peg$FAILED) {
                    if (input.charCodeAt(peg$currPos) === 41) {
                      s6 = peg$c18;
                      peg$currPos++;
                    } else {
                      s6 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c19); }
                    }
                    if (s6 !== peg$FAILED) {
                      peg$savedPos = s0;
                      s1 = peg$c42(s4);
                      s0 = s1;
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
          if (s0 === peg$FAILED) {
            s0 = peg$currPos;
            if (input.charCodeAt(peg$currPos) === 40) {
              s1 = peg$c16;
              peg$currPos++;
            } else {
              s1 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c17); }
            }
            if (s1 !== peg$FAILED) {
              s2 = peg$parse__();
              if (s2 !== peg$FAILED) {
                s3 = peg$parseScope();
                if (s3 !== peg$FAILED) {
                  s4 = peg$parse__();
                  if (s4 !== peg$FAILED) {
                    if (input.charCodeAt(peg$currPos) === 41) {
                      s5 = peg$c18;
                      peg$currPos++;
                    } else {
                      s5 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c19); }
                    }
                    if (s5 !== peg$FAILED) {
                      peg$savedPos = s0;
                      s1 = peg$c43(s3);
                      s0 = s1;
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
            if (s0 === peg$FAILED) {
              s0 = peg$parseOperator();
              if (s0 === peg$FAILED) {
                s0 = peg$currPos;
                s1 = peg$parseOpAssignment();
                if (s1 !== peg$FAILED) {
                  s2 = peg$currPos;
                  peg$silentFails++;
                  s3 = peg$parseEndOfOp();
                  peg$silentFails--;
                  if (s3 !== peg$FAILED) {
                    peg$currPos = s2;
                    s2 = void 0;
                  } else {
                    s2 = peg$FAILED;
                  }
                  if (s2 !== peg$FAILED) {
                    peg$savedPos = s0;
                    s1 = peg$c44(s1);
                    s0 = s1;
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
                if (s0 === peg$FAILED) {
                  s0 = peg$currPos;
                  s1 = peg$currPos;
                  peg$silentFails++;
                  s2 = peg$currPos;
                  s3 = peg$parseFunction();
                  if (s3 !== peg$FAILED) {
                    s4 = peg$parseEndOfOp();
                    if (s4 !== peg$FAILED) {
                      s3 = [s3, s4];
                      s2 = s3;
                    } else {
                      peg$currPos = s2;
                      s2 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s2;
                    s2 = peg$FAILED;
                  }
                  peg$silentFails--;
                  if (s2 === peg$FAILED) {
                    s1 = void 0;
                  } else {
                    peg$currPos = s1;
                    s1 = peg$FAILED;
                  }
                  if (s1 !== peg$FAILED) {
                    s2 = peg$parseAggregation();
                    if (s2 !== peg$FAILED) {
                      s3 = peg$currPos;
                      peg$silentFails++;
                      s4 = peg$parseEndOfOp();
                      peg$silentFails--;
                      if (s4 !== peg$FAILED) {
                        peg$currPos = s3;
                        s3 = void 0;
                      } else {
                        s3 = peg$FAILED;
                      }
                      if (s3 !== peg$FAILED) {
                        peg$savedPos = s0;
                        s1 = peg$c44(s2);
                        s0 = s1;
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                  if (s0 === peg$FAILED) {
                    s0 = peg$currPos;
                    if (input.substr(peg$currPos, 6) === peg$c45) {
                      s1 = peg$c45;
                      peg$currPos += 6;
                    } else {
                      s1 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c46); }
                    }
                    if (s1 !== peg$FAILED) {
                      s2 = peg$parse_();
                      if (s2 !== peg$FAILED) {
                        s3 = peg$parseSearchBoolean();
                        if (s3 !== peg$FAILED) {
                          peg$savedPos = s0;
                          s1 = peg$c47(s3);
                          s0 = s1;
                        } else {
                          peg$currPos = s0;
                          s0 = peg$FAILED;
                        }
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                    if (s0 === peg$FAILED) {
                      s0 = peg$currPos;
                      s1 = peg$parseSearchBoolean();
                      if (s1 !== peg$FAILED) {
                        peg$savedPos = s0;
                        s1 = peg$c48(s1);
                      }
                      s0 = s1;
                      if (s0 === peg$FAILED) {
                        s0 = peg$currPos;
                        s1 = peg$parseCast();
                        if (s1 !== peg$FAILED) {
                          peg$savedPos = s0;
                          s1 = peg$c49(s1);
                        }
                        s0 = s1;
                        if (s0 === peg$FAILED) {
                          s0 = peg$currPos;
                          s1 = peg$parseConditionalExpr();
                          if (s1 !== peg$FAILED) {
                            peg$savedPos = s0;
                            s1 = peg$c48(s1);
                          }
                          s0 = s1;
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }

    return s0;
  }

  function peg$parseEndOfOp() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      s2 = peg$parsePipe();
      if (s2 === peg$FAILED) {
        s2 = peg$parseSearchKeywordGuard();
        if (s2 === peg$FAILED) {
          if (input.substr(peg$currPos, 2) === peg$c50) {
            s2 = peg$c50;
            peg$currPos += 2;
          } else {
            s2 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c51); }
          }
          if (s2 === peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 41) {
              s2 = peg$c18;
              peg$currPos++;
            } else {
              s2 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c19); }
            }
            if (s2 === peg$FAILED) {
              s2 = peg$parseEOF();
            }
          }
        }
      }
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePipe() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 124) {
      s1 = peg$c52;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c53); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      if (input.charCodeAt(peg$currPos) === 123) {
        s3 = peg$c54;
        peg$currPos++;
      } else {
        s3 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c55); }
      }
      if (s3 === peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 91) {
          s3 = peg$c56;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c57); }
        }
      }
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseLeg() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 2) === peg$c50) {
        s2 = peg$c50;
        peg$currPos += 2;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c51); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseSeq();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c1(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSwitchLeg() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseCase();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseLeg();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c58(s2, s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseCase() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c59) {
      s1 = peg$c59;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c60); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseConditionalExpr();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c61(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 7) === peg$c62) {
        s1 = peg$c62;
        peg$currPos += 7;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c63); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c64();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseFromLeg() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseFromSource();
      if (s2 !== peg$FAILED) {
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.substr(peg$currPos, 2) === peg$c50) {
            s5 = peg$c50;
            peg$currPos += 2;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c51); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseSeq();
              if (s7 !== peg$FAILED) {
                s4 = [s4, s5, s6, s7];
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
        if (s3 === peg$FAILED) {
          s3 = null;
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c65(s2, s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFromSource() {
    var s0;

    s0 = peg$parseFile();
    if (s0 === peg$FAILED) {
      s0 = peg$parseGet();
      if (s0 === peg$FAILED) {
        s0 = peg$parsePool();
        if (s0 === peg$FAILED) {
          s0 = peg$parsePassOp();
        }
      }
    }

    return s0;
  }

  function peg$parseExprGuard() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      s3 = peg$currPos;
      peg$silentFails++;
      if (input.substr(peg$currPos, 2) === peg$c50) {
        s4 = peg$c50;
        peg$currPos += 2;
      } else {
        s4 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c51); }
      }
      peg$silentFails--;
      if (s4 === peg$FAILED) {
        s3 = void 0;
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      if (s3 !== peg$FAILED) {
        s4 = peg$parseComparator();
        if (s4 !== peg$FAILED) {
          s3 = [s3, s4];
          s2 = s3;
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 === peg$FAILED) {
        s2 = peg$parseAdditiveOperator();
        if (s2 === peg$FAILED) {
          s2 = peg$parseMultiplicativeOperator();
          if (s2 === peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 58) {
              s2 = peg$c20;
              peg$currPos++;
            } else {
              s2 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c21); }
            }
            if (s2 === peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 40) {
                s2 = peg$c16;
                peg$currPos++;
              } else {
                s2 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c17); }
              }
              if (s2 === peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 91) {
                  s2 = peg$c56;
                  peg$currPos++;
                } else {
                  s2 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c57); }
                }
                if (s2 === peg$FAILED) {
                  if (input.charCodeAt(peg$currPos) === 126) {
                    s2 = peg$c66;
                    peg$currPos++;
                  } else {
                    s2 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c67); }
                  }
                }
              }
            }
          }
        }
      }
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseComparator() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c68) {
      s1 = peg$c68;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c69); }
    }
    if (s1 === peg$FAILED) {
      if (input.substr(peg$currPos, 2) === peg$c70) {
        s1 = peg$c70;
        peg$currPos += 2;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c71); }
      }
      if (s1 === peg$FAILED) {
        s1 = peg$currPos;
        if (input.substr(peg$currPos, 2) === peg$c72) {
          s2 = peg$c72;
          peg$currPos += 2;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c73); }
        }
        if (s2 !== peg$FAILED) {
          s3 = peg$currPos;
          peg$silentFails++;
          s4 = peg$parseIdentifierRest();
          peg$silentFails--;
          if (s4 === peg$FAILED) {
            s3 = void 0;
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
          if (s3 !== peg$FAILED) {
            s2 = [s2, s3];
            s1 = s2;
          } else {
            peg$currPos = s1;
            s1 = peg$FAILED;
          }
        } else {
          peg$currPos = s1;
          s1 = peg$FAILED;
        }
        if (s1 === peg$FAILED) {
          if (input.substr(peg$currPos, 2) === peg$c74) {
            s1 = peg$c74;
            peg$currPos += 2;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c75); }
          }
          if (s1 === peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 60) {
              s1 = peg$c76;
              peg$currPos++;
            } else {
              s1 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c77); }
            }
            if (s1 === peg$FAILED) {
              if (input.substr(peg$currPos, 2) === peg$c78) {
                s1 = peg$c78;
                peg$currPos += 2;
              } else {
                s1 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c79); }
              }
              if (s1 === peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 62) {
                  s1 = peg$c80;
                  peg$currPos++;
                } else {
                  s1 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c81); }
                }
              }
            }
          }
        }
      }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c82();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseSearchBoolean() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseSearchAnd();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseSearchOrTerm();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseSearchOrTerm();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c83(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSearchOrTerm() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseOrToken();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseSearchAnd();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c84(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSearchAnd() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseSearchFactor();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$currPos;
      s5 = peg$parse_();
      if (s5 !== peg$FAILED) {
        s6 = peg$parseAndToken();
        if (s6 !== peg$FAILED) {
          s5 = [s5, s6];
          s4 = s5;
        } else {
          peg$currPos = s4;
          s4 = peg$FAILED;
        }
      } else {
        peg$currPos = s4;
        s4 = peg$FAILED;
      }
      if (s4 === peg$FAILED) {
        s4 = null;
      }
      if (s4 !== peg$FAILED) {
        s5 = peg$parse_();
        if (s5 !== peg$FAILED) {
          s6 = peg$currPos;
          peg$silentFails++;
          s7 = peg$parseOrToken();
          if (s7 === peg$FAILED) {
            s7 = peg$parseSearchKeywordGuard();
          }
          peg$silentFails--;
          if (s7 === peg$FAILED) {
            s6 = void 0;
          } else {
            peg$currPos = s6;
            s6 = peg$FAILED;
          }
          if (s6 !== peg$FAILED) {
            s7 = peg$parseSearchFactor();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c85(s1, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$currPos;
        s5 = peg$parse_();
        if (s5 !== peg$FAILED) {
          s6 = peg$parseAndToken();
          if (s6 !== peg$FAILED) {
            s5 = [s5, s6];
            s4 = s5;
          } else {
            peg$currPos = s4;
            s4 = peg$FAILED;
          }
        } else {
          peg$currPos = s4;
          s4 = peg$FAILED;
        }
        if (s4 === peg$FAILED) {
          s4 = null;
        }
        if (s4 !== peg$FAILED) {
          s5 = peg$parse_();
          if (s5 !== peg$FAILED) {
            s6 = peg$currPos;
            peg$silentFails++;
            s7 = peg$parseOrToken();
            if (s7 === peg$FAILED) {
              s7 = peg$parseSearchKeywordGuard();
            }
            peg$silentFails--;
            if (s7 === peg$FAILED) {
              s6 = void 0;
            } else {
              peg$currPos = s6;
              s6 = peg$FAILED;
            }
            if (s6 !== peg$FAILED) {
              s7 = peg$parseSearchFactor();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c85(s1, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c83(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSearchKeywordGuard() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parseFromSource();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.substr(peg$currPos, 2) === peg$c50) {
          s3 = peg$c50;
          peg$currPos += 2;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c51); }
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            s1 = [s1, s2, s3, s4];
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseCase();
      if (s1 !== peg$FAILED) {
        s2 = peg$parse__();
        if (s2 !== peg$FAILED) {
          s1 = [s1, s2];
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseSearchFactor() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$currPos;
    s2 = peg$parseNotToken();
    if (s2 !== peg$FAILED) {
      s3 = peg$parse_();
      if (s3 !== peg$FAILED) {
        s2 = [s2, s3];
        s1 = s2;
      } else {
        peg$currPos = s1;
        s1 = peg$FAILED;
      }
    } else {
      peg$currPos = s1;
      s1 = peg$FAILED;
    }
    if (s1 === peg$FAILED) {
      s1 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 33) {
        s2 = peg$c86;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c87); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s2 = [s2, s3];
          s1 = s2;
        } else {
          peg$currPos = s1;
          s1 = peg$FAILED;
        }
      } else {
        peg$currPos = s1;
        s1 = peg$FAILED;
      }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseSearchFactor();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c88(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 40) {
        s1 = peg$c16;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c17); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parse__();
        if (s2 !== peg$FAILED) {
          s3 = peg$parseSearchBoolean();
          if (s3 !== peg$FAILED) {
            s4 = peg$parse__();
            if (s4 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 41) {
                s5 = peg$c18;
                peg$currPos++;
              } else {
                s5 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c19); }
              }
              if (s5 !== peg$FAILED) {
                peg$savedPos = s0;
                s1 = peg$c61(s3);
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$parseSearchExpr();
      }
    }

    return s0;
  }

  function peg$parseSearchExpr() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$parseRegexp();
    if (s0 === peg$FAILED) {
      s0 = peg$parseGlob();
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        s1 = peg$parseSearchValue();
        if (s1 !== peg$FAILED) {
          s2 = peg$currPos;
          peg$silentFails++;
          s3 = peg$parseExprGuard();
          peg$silentFails--;
          if (s3 === peg$FAILED) {
            s2 = void 0;
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
          if (s2 === peg$FAILED) {
            s2 = peg$currPos;
            peg$silentFails++;
            s3 = peg$currPos;
            s4 = peg$parse_();
            if (s4 !== peg$FAILED) {
              s5 = peg$parseGlob();
              if (s5 !== peg$FAILED) {
                s4 = [s4, s5];
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
            peg$silentFails--;
            if (s3 !== peg$FAILED) {
              peg$currPos = s2;
              s2 = void 0;
            } else {
              s2 = peg$FAILED;
            }
          }
          if (s2 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c89(s1);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          if (input.charCodeAt(peg$currPos) === 42) {
            s1 = peg$c90;
            peg$currPos++;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c91); }
          }
          if (s1 !== peg$FAILED) {
            s2 = peg$currPos;
            peg$silentFails++;
            s3 = peg$parseExprGuard();
            peg$silentFails--;
            if (s3 === peg$FAILED) {
              s2 = void 0;
            } else {
              peg$currPos = s2;
              s2 = peg$FAILED;
            }
            if (s2 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c92();
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
          if (s0 === peg$FAILED) {
            s0 = peg$parseSearchPredicate();
          }
        }
      }
    }

    return s0;
  }

  function peg$parseSearchPredicate() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parseAdditiveExpr();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseComparator();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseAdditiveExpr();
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c93(s1, s3, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseFunction();
      if (s1 !== peg$FAILED) {
        s2 = [];
        s3 = peg$parseDeref();
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          s3 = peg$parseDeref();
        }
        if (s2 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c94(s1, s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseSearchValue() {
    var s0, s1, s2;

    s0 = peg$parseLiteral();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$currPos;
      peg$silentFails++;
      s2 = peg$parseRegexpPattern();
      peg$silentFails--;
      if (s2 === peg$FAILED) {
        s1 = void 0;
      } else {
        peg$currPos = s1;
        s1 = peg$FAILED;
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseKeyWord();
        if (s2 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c95(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseGlob() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseGlobPattern();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c96(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseRegexp() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseRegexpPattern();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c97(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseAggregation() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parseSummarize();
    if (s1 === peg$FAILED) {
      s1 = null;
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseGroupByKeys();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseLimitArg();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c98(s2, s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseSummarize();
      if (s1 === peg$FAILED) {
        s1 = null;
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseAggAssignments();
        if (s2 !== peg$FAILED) {
          s3 = peg$currPos;
          s4 = peg$parse_();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseGroupByKeys();
            if (s5 !== peg$FAILED) {
              s4 = [s4, s5];
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
          if (s3 === peg$FAILED) {
            s3 = null;
          }
          if (s3 !== peg$FAILED) {
            s4 = peg$parseLimitArg();
            if (s4 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c99(s2, s3, s4);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseSummarize() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 9) === peg$c100) {
      s1 = peg$c100;
      peg$currPos += 9;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c101); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseGroupByKeys() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseByToken();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseFlexAssignments();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c102(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseLimitArg() {
    var s0, s1, s2, s3, s4, s5, s6;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 4) === peg$c103) {
        s2 = peg$c103;
        peg$currPos += 4;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c104); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          if (input.substr(peg$currPos, 6) === peg$c105) {
            s4 = peg$c105;
            peg$currPos += 6;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c106); }
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parse_();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseUInt();
              if (s6 !== peg$FAILED) {
                peg$savedPos = s0;
                s1 = peg$c107(s6);
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$c108;
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c109();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseFlexAssignment() {
    var s0, s1;

    s0 = peg$parseAssignment();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseConditionalExpr();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c110(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseFlexAssignments() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseFlexAssignment();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s5 = peg$c26;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c27); }
        }
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseFlexAssignment();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c111(s1, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 44) {
            s5 = peg$c26;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c27); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseFlexAssignment();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c111(s1, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c3(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseAggAssignment() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parseDerefExpr();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.substr(peg$currPos, 2) === peg$c112) {
          s3 = peg$c112;
          peg$currPos += 2;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c113); }
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseAgg();
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c114(s1, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseAgg();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c115(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseAgg() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10, s11, s12;

    s0 = peg$currPos;
    s1 = peg$currPos;
    peg$silentFails++;
    s2 = peg$parseFuncGuard();
    peg$silentFails--;
    if (s2 === peg$FAILED) {
      s1 = void 0;
    } else {
      peg$currPos = s1;
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseAggName();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 40) {
            s4 = peg$c16;
            peg$currPos++;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c17); }
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parse__();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseOverExpr();
              if (s6 === peg$FAILED) {
                s6 = peg$parseConditionalExpr();
              }
              if (s6 === peg$FAILED) {
                s6 = null;
              }
              if (s6 !== peg$FAILED) {
                s7 = peg$parse__();
                if (s7 !== peg$FAILED) {
                  if (input.charCodeAt(peg$currPos) === 41) {
                    s8 = peg$c18;
                    peg$currPos++;
                  } else {
                    s8 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c19); }
                  }
                  if (s8 !== peg$FAILED) {
                    s9 = peg$currPos;
                    peg$silentFails++;
                    s10 = peg$currPos;
                    s11 = peg$parse__();
                    if (s11 !== peg$FAILED) {
                      if (input.charCodeAt(peg$currPos) === 46) {
                        s12 = peg$c116;
                        peg$currPos++;
                      } else {
                        s12 = peg$FAILED;
                        if (peg$silentFails === 0) { peg$fail(peg$c117); }
                      }
                      if (s12 !== peg$FAILED) {
                        s11 = [s11, s12];
                        s10 = s11;
                      } else {
                        peg$currPos = s10;
                        s10 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s10;
                      s10 = peg$FAILED;
                    }
                    peg$silentFails--;
                    if (s10 === peg$FAILED) {
                      s9 = void 0;
                    } else {
                      peg$currPos = s9;
                      s9 = peg$FAILED;
                    }
                    if (s9 !== peg$FAILED) {
                      s10 = peg$parseWhereClause();
                      if (s10 === peg$FAILED) {
                        s10 = null;
                      }
                      if (s10 !== peg$FAILED) {
                        peg$savedPos = s0;
                        s1 = peg$c118(s2, s6, s10);
                        s0 = s1;
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseAggName() {
    var s0;

    s0 = peg$parseIdentifierName();
    if (s0 === peg$FAILED) {
      s0 = peg$parseAndToken();
      if (s0 === peg$FAILED) {
        s0 = peg$parseOrToken();
      }
    }

    return s0;
  }

  function peg$parseWhereClause() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 5) === peg$c119) {
        s2 = peg$c119;
        peg$currPos += 5;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c120); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseLogicalOrExpr();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c61(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseAggAssignments() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseAggAssignment();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s5 = peg$c26;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c27); }
        }
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseAggAssignment();
            if (s7 !== peg$FAILED) {
              s4 = [s4, s5, s6, s7];
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 44) {
            s5 = peg$c26;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c27); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseAggAssignment();
              if (s7 !== peg$FAILED) {
                s4 = [s4, s5, s6, s7];
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c121(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseOperator() {
    var s0;

    s0 = peg$parseAssertOp();
    if (s0 === peg$FAILED) {
      s0 = peg$parseSortOp();
      if (s0 === peg$FAILED) {
        s0 = peg$parseTopOp();
        if (s0 === peg$FAILED) {
          s0 = peg$parseCutOp();
          if (s0 === peg$FAILED) {
            s0 = peg$parseDropOp();
            if (s0 === peg$FAILED) {
              s0 = peg$parseHeadOp();
              if (s0 === peg$FAILED) {
                s0 = peg$parseTailOp();
                if (s0 === peg$FAILED) {
                  s0 = peg$parseWhereOp();
                  if (s0 === peg$FAILED) {
                    s0 = peg$parseUniqOp();
                    if (s0 === peg$FAILED) {
                      s0 = peg$parsePutOp();
                      if (s0 === peg$FAILED) {
                        s0 = peg$parseRenameOp();
                        if (s0 === peg$FAILED) {
                          s0 = peg$parseFuseOp();
                          if (s0 === peg$FAILED) {
                            s0 = peg$parseShapeOp();
                            if (s0 === peg$FAILED) {
                              s0 = peg$parseJoinOp();
                              if (s0 === peg$FAILED) {
                                s0 = peg$parseSampleOp();
                                if (s0 === peg$FAILED) {
                                  s0 = peg$parseSQLOp();
                                  if (s0 === peg$FAILED) {
                                    s0 = peg$parseFromOp();
                                    if (s0 === peg$FAILED) {
                                      s0 = peg$parsePassOp();
                                      if (s0 === peg$FAILED) {
                                        s0 = peg$parseExplodeOp();
                                        if (s0 === peg$FAILED) {
                                          s0 = peg$parseMergeOp();
                                          if (s0 === peg$FAILED) {
                                            s0 = peg$parseOverOp();
                                            if (s0 === peg$FAILED) {
                                              s0 = peg$parseYieldOp();
                                              if (s0 === peg$FAILED) {
                                                s0 = peg$parseLoadOp();
                                              }
                                            }
                                          }
                                        }
                                      }
                                    }
                                  }
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }

    return s0;
  }

  function peg$parseAssertOp() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 6) === peg$c122) {
      s1 = peg$c122;
      peg$currPos += 6;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c123); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$currPos;
        s4 = peg$parseConditionalExpr();
        if (s4 !== peg$FAILED) {
          peg$savedPos = s3;
          s4 = peg$c124(s4);
        }
        s3 = s4;
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c125(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSortOp() {
    var s0, s1, s2, s3, s4, s5, s6;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c126) {
      s1 = peg$c126;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c127); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseEOKW();
      peg$silentFails--;
      if (s3 !== peg$FAILED) {
        peg$currPos = s2;
        s2 = void 0;
      } else {
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseSortArgs();
        if (s3 !== peg$FAILED) {
          s4 = peg$currPos;
          s5 = peg$parse_();
          if (s5 !== peg$FAILED) {
            s6 = peg$parseExprs();
            if (s6 !== peg$FAILED) {
              peg$savedPos = s4;
              s5 = peg$c128(s3, s6);
              s4 = s5;
            } else {
              peg$currPos = s4;
              s4 = peg$FAILED;
            }
          } else {
            peg$currPos = s4;
            s4 = peg$FAILED;
          }
          if (s4 === peg$FAILED) {
            s4 = null;
          }
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c129(s3, s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSortArgs() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = [];
    s2 = peg$currPos;
    s3 = peg$parse_();
    if (s3 !== peg$FAILED) {
      s4 = peg$parseSortArg();
      if (s4 !== peg$FAILED) {
        peg$savedPos = s2;
        s3 = peg$c44(s4);
        s2 = s3;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
    } else {
      peg$currPos = s2;
      s2 = peg$FAILED;
    }
    while (s2 !== peg$FAILED) {
      s1.push(s2);
      s2 = peg$currPos;
      s3 = peg$parse_();
      if (s3 !== peg$FAILED) {
        s4 = peg$parseSortArg();
        if (s4 !== peg$FAILED) {
          peg$savedPos = s2;
          s3 = peg$c44(s4);
          s2 = s3;
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c130(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseSortArg() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c131) {
      s1 = peg$c131;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c132); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c133();
    }
    s0 = s1;
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 6) === peg$c134) {
        s1 = peg$c134;
        peg$currPos += 6;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c135); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parse_();
        if (s2 !== peg$FAILED) {
          s3 = peg$currPos;
          if (input.substr(peg$currPos, 5) === peg$c136) {
            s4 = peg$c136;
            peg$currPos += 5;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c137); }
          }
          if (s4 === peg$FAILED) {
            if (input.substr(peg$currPos, 4) === peg$c138) {
              s4 = peg$c138;
              peg$currPos += 4;
            } else {
              s4 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c139); }
            }
          }
          if (s4 !== peg$FAILED) {
            peg$savedPos = s3;
            s4 = peg$c82();
          }
          s3 = s4;
          if (s3 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c140(s3);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseTopOp() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 3) === peg$c141) {
      s1 = peg$c141;
      peg$currPos += 3;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c142); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseEOKW();
      peg$silentFails--;
      if (s3 !== peg$FAILED) {
        peg$currPos = s2;
        s2 = void 0;
      } else {
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$currPos;
        s4 = peg$parse_();
        if (s4 !== peg$FAILED) {
          s5 = peg$parseUInt();
          if (s5 !== peg$FAILED) {
            peg$savedPos = s3;
            s4 = peg$c143(s5);
            s3 = s4;
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
        if (s3 === peg$FAILED) {
          s3 = null;
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$currPos;
          s5 = peg$parse_();
          if (s5 !== peg$FAILED) {
            if (input.substr(peg$currPos, 6) === peg$c144) {
              s6 = peg$c144;
              peg$currPos += 6;
            } else {
              s6 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c145); }
            }
            if (s6 !== peg$FAILED) {
              s5 = [s5, s6];
              s4 = s5;
            } else {
              peg$currPos = s4;
              s4 = peg$FAILED;
            }
          } else {
            peg$currPos = s4;
            s4 = peg$FAILED;
          }
          if (s4 === peg$FAILED) {
            s4 = null;
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$currPos;
            s6 = peg$parse_();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseFieldExprs();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s5;
                s6 = peg$c146(s3, s4, s7);
                s5 = s6;
              } else {
                peg$currPos = s5;
                s5 = peg$FAILED;
              }
            } else {
              peg$currPos = s5;
              s5 = peg$FAILED;
            }
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c147(s3, s4, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseCutOp() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 3) === peg$c148) {
      s1 = peg$c148;
      peg$currPos += 3;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c149); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseFlexAssignments();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c150(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseDropOp() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c151) {
      s1 = peg$c151;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c152); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseFieldExprs();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c153(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseHeadOp() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c154) {
      s1 = peg$c154;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c155); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$currPos;
        peg$silentFails++;
        s4 = peg$parseEndOfOp();
        peg$silentFails--;
        if (s4 === peg$FAILED) {
          s3 = void 0;
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parseConditionalExpr();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c156(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 4) === peg$c154) {
        s1 = peg$c154;
        peg$currPos += 4;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c155); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c157();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseTailOp() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c158) {
      s1 = peg$c158;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c159); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$currPos;
        peg$silentFails++;
        s4 = peg$parseEndOfOp();
        peg$silentFails--;
        if (s4 === peg$FAILED) {
          s3 = void 0;
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parseConditionalExpr();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c160(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 4) === peg$c158) {
        s1 = peg$c158;
        peg$currPos += 4;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c159); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c161();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseWhereOp() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5) === peg$c119) {
      s1 = peg$c119;
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c120); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseConditionalExpr();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c162(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseUniqOp() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c163) {
      s1 = peg$c163;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c164); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        if (input.substr(peg$currPos, 2) === peg$c165) {
          s3 = peg$c165;
          peg$currPos += 2;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c166); }
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c167();
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 4) === peg$c163) {
        s1 = peg$c163;
        peg$currPos += 4;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c164); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c168();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parsePutOp() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 3) === peg$c169) {
      s1 = peg$c169;
      peg$currPos += 3;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c170); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseAssignments();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c171(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseRenameOp() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 6) === peg$c172) {
      s1 = peg$c172;
      peg$currPos += 6;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c173); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseAssignment();
        if (s3 !== peg$FAILED) {
          s4 = [];
          s5 = peg$currPos;
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 44) {
              s7 = peg$c26;
              peg$currPos++;
            } else {
              s7 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c27); }
            }
            if (s7 !== peg$FAILED) {
              s8 = peg$parse__();
              if (s8 !== peg$FAILED) {
                s9 = peg$parseAssignment();
                if (s9 !== peg$FAILED) {
                  peg$savedPos = s5;
                  s6 = peg$c174(s3, s9);
                  s5 = s6;
                } else {
                  peg$currPos = s5;
                  s5 = peg$FAILED;
                }
              } else {
                peg$currPos = s5;
                s5 = peg$FAILED;
              }
            } else {
              peg$currPos = s5;
              s5 = peg$FAILED;
            }
          } else {
            peg$currPos = s5;
            s5 = peg$FAILED;
          }
          while (s5 !== peg$FAILED) {
            s4.push(s5);
            s5 = peg$currPos;
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 44) {
                s7 = peg$c26;
                peg$currPos++;
              } else {
                s7 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c27); }
              }
              if (s7 !== peg$FAILED) {
                s8 = peg$parse__();
                if (s8 !== peg$FAILED) {
                  s9 = peg$parseAssignment();
                  if (s9 !== peg$FAILED) {
                    peg$savedPos = s5;
                    s6 = peg$c174(s3, s9);
                    s5 = s6;
                  } else {
                    peg$currPos = s5;
                    s5 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s5;
                  s5 = peg$FAILED;
                }
              } else {
                peg$currPos = s5;
                s5 = peg$FAILED;
              }
            } else {
              peg$currPos = s5;
              s5 = peg$FAILED;
            }
          }
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c175(s3, s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFuseOp() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c176) {
      s1 = peg$c176;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c177); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 40) {
          s5 = peg$c16;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c17); }
        }
        if (s5 !== peg$FAILED) {
          s4 = [s4, s5];
          s3 = s4;
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$currPos;
        peg$silentFails++;
        s4 = peg$parseEOKW();
        peg$silentFails--;
        if (s4 !== peg$FAILED) {
          peg$currPos = s3;
          s3 = void 0;
        } else {
          s3 = peg$FAILED;
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c178();
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseShapeOp() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5) === peg$c179) {
      s1 = peg$c179;
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c180); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 40) {
          s5 = peg$c16;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c17); }
        }
        if (s5 !== peg$FAILED) {
          s4 = [s4, s5];
          s3 = s4;
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$currPos;
        peg$silentFails++;
        s4 = peg$parseEOKW();
        peg$silentFails--;
        if (s4 !== peg$FAILED) {
          peg$currPos = s3;
          s3 = void 0;
        } else {
          s3 = peg$FAILED;
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c181();
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseJoinOp() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10, s11;

    s0 = peg$currPos;
    s1 = peg$parseJoinStyle();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 4) === peg$c182) {
        s2 = peg$c182;
        peg$currPos += 4;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c183); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseJoinRightInput();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseON();
          if (s4 !== peg$FAILED) {
            s5 = peg$parse_();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseJoinKey();
              if (s6 !== peg$FAILED) {
                s7 = peg$currPos;
                s8 = peg$parse__();
                if (s8 !== peg$FAILED) {
                  if (input.charCodeAt(peg$currPos) === 61) {
                    s9 = peg$c8;
                    peg$currPos++;
                  } else {
                    s9 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c9); }
                  }
                  if (s9 !== peg$FAILED) {
                    s10 = peg$parse__();
                    if (s10 !== peg$FAILED) {
                      s11 = peg$parseJoinKey();
                      if (s11 !== peg$FAILED) {
                        s8 = [s8, s9, s10, s11];
                        s7 = s8;
                      } else {
                        peg$currPos = s7;
                        s7 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s7;
                      s7 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s7;
                    s7 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s7;
                  s7 = peg$FAILED;
                }
                if (s7 === peg$FAILED) {
                  s7 = null;
                }
                if (s7 !== peg$FAILED) {
                  s8 = peg$currPos;
                  s9 = peg$parse_();
                  if (s9 !== peg$FAILED) {
                    s10 = peg$parseFlexAssignments();
                    if (s10 !== peg$FAILED) {
                      s9 = [s9, s10];
                      s8 = s9;
                    } else {
                      peg$currPos = s8;
                      s8 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s8;
                    s8 = peg$FAILED;
                  }
                  if (s8 === peg$FAILED) {
                    s8 = null;
                  }
                  if (s8 !== peg$FAILED) {
                    peg$savedPos = s0;
                    s1 = peg$c184(s1, s3, s6, s7, s8);
                    s0 = s1;
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseJoinStyle() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c185) {
      s1 = peg$c185;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c186); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c187();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 5) === peg$c188) {
        s1 = peg$c188;
        peg$currPos += 5;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c189); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parse_();
        if (s2 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c190();
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        if (input.substr(peg$currPos, 4) === peg$c191) {
          s1 = peg$c191;
          peg$currPos += 4;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c192); }
        }
        if (s1 !== peg$FAILED) {
          s2 = peg$parse_();
          if (s2 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c193();
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          if (input.substr(peg$currPos, 5) === peg$c194) {
            s1 = peg$c194;
            peg$currPos += 5;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c195); }
          }
          if (s1 !== peg$FAILED) {
            s2 = peg$parse_();
            if (s2 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c196();
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
          if (s0 === peg$FAILED) {
            s0 = peg$currPos;
            s1 = peg$c108;
            if (s1 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c190();
            }
            s0 = s1;
          }
        }
      }
    }

    return s0;
  }

  function peg$parseJoinRightInput() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 40) {
        s2 = peg$c16;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c17); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseSeq();
          if (s4 !== peg$FAILED) {
            s5 = peg$parse__();
            if (s5 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 41) {
                s6 = peg$c18;
                peg$currPos++;
              } else {
                s6 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c19); }
              }
              if (s6 !== peg$FAILED) {
                s7 = peg$parse__();
                if (s7 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c197(s4);
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parse_();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c64();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseJoinKey() {
    var s0, s1, s2, s3;

    s0 = peg$parseDerefExpr();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 40) {
        s1 = peg$c16;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c17); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseConditionalExpr();
        if (s2 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 41) {
            s3 = peg$c18;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c19); }
          }
          if (s3 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c61(s2);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseSampleOp() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 6) === peg$c198) {
      s1 = peg$c198;
      peg$currPos += 6;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c199); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseEOKW();
      peg$silentFails--;
      if (s3 !== peg$FAILED) {
        peg$currPos = s2;
        s2 = void 0;
      } else {
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseSampleExpr();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c200(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseOpAssignment() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseAssignments();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c201(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseSampleExpr() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseDerefExpr();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c202(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$c108;
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c203();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseLoadOp() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c204) {
      s1 = peg$c204;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c205); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parsePoolNameString();
        if (s3 !== peg$FAILED) {
          s4 = peg$parsePoolBranch();
          if (s4 === peg$FAILED) {
            s4 = null;
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parseAuthorArg();
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              s6 = peg$parseMessageArg();
              if (s6 === peg$FAILED) {
                s6 = null;
              }
              if (s6 !== peg$FAILED) {
                s7 = peg$parseMetaArg();
                if (s7 === peg$FAILED) {
                  s7 = null;
                }
                if (s7 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c206(s3, s4, s5, s6, s7);
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseAuthorArg() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 6) === peg$c207) {
        s2 = peg$c207;
        peg$currPos += 6;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c208); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseQuotedString();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c209(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseMessageArg() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 7) === peg$c210) {
        s2 = peg$c210;
        peg$currPos += 7;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c211); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseQuotedString();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c209(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseMetaArg() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 4) === peg$c212) {
        s2 = peg$c212;
        peg$currPos += 4;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c213); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseQuotedString();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c209(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePoolBranch() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 64) {
      s1 = peg$c214;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c215); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parsePoolIdentifier();
      if (s2 === peg$FAILED) {
        s2 = peg$parseQuotedString();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c216(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFromOp() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseFromAny();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c217(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseFromAny() {
    var s0;

    s0 = peg$parseFile();
    if (s0 === peg$FAILED) {
      s0 = peg$parseGet();
      if (s0 === peg$FAILED) {
        s0 = peg$parseFrom();
      }
    }

    return s0;
  }

  function peg$parseFile() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c218) {
      s1 = peg$c218;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c219); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parsePath();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseFormatArg();
          if (s4 === peg$FAILED) {
            s4 = null;
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parseSortKeyArg();
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c220(s3, s4, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFrom() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c40) {
      s1 = peg$c40;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c41); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parsePoolBody();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c221(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePool() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c222) {
      s1 = peg$c222;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c223); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parsePoolBody();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c221(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePoolBody() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parsePoolSpec();
    if (s1 !== peg$FAILED) {
      s2 = peg$parsePoolAt();
      if (s2 === peg$FAILED) {
        s2 = null;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c224(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseGet() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 3) === peg$c225) {
      s1 = peg$c225;
      peg$currPos += 3;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c226); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseURL();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseFormatArg();
          if (s4 === peg$FAILED) {
            s4 = null;
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parseSortKeyArg();
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c227(s3, s4, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseURL() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5) === peg$c228) {
      s1 = peg$c228;
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c229); }
    }
    if (s1 === peg$FAILED) {
      if (input.substr(peg$currPos, 6) === peg$c230) {
        s1 = peg$c230;
        peg$currPos += 6;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c231); }
      }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parsePath();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c82();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePath() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parseQuotedString();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c5(s1);
    }
    s0 = s1;
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = [];
      if (peg$c232.test(input.charAt(peg$currPos))) {
        s2 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c233); }
      }
      if (s2 !== peg$FAILED) {
        while (s2 !== peg$FAILED) {
          s1.push(s2);
          if (peg$c232.test(input.charAt(peg$currPos))) {
            s2 = input.charAt(peg$currPos);
            peg$currPos++;
          } else {
            s2 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c233); }
          }
        }
      } else {
        s1 = peg$FAILED;
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c82();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parsePoolAt() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 2) === peg$c234) {
        s2 = peg$c234;
        peg$currPos += 2;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c235); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseKSUID();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c236(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseKSUID() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = [];
    if (peg$c237.test(input.charAt(peg$currPos))) {
      s2 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s2 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c238); }
    }
    if (s2 !== peg$FAILED) {
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        if (peg$c237.test(input.charAt(peg$currPos))) {
          s2 = input.charAt(peg$currPos);
          peg$currPos++;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c238); }
        }
      }
    } else {
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c82();
    }
    s0 = s1;

    return s0;
  }

  function peg$parsePoolSpec() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parsePoolName();
    if (s1 !== peg$FAILED) {
      s2 = peg$parsePoolCommit();
      if (s2 === peg$FAILED) {
        s2 = null;
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parsePoolMeta();
        if (s3 === peg$FAILED) {
          s3 = null;
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parseTapArg();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c239(s1, s2, s3, s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parsePoolMeta();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c240(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parsePoolCommit() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 64) {
      s1 = peg$c214;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c215); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parsePoolNameString();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c241(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePoolMeta() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 58) {
      s1 = peg$c20;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c21); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parsePoolIdentifier();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c242(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePoolName() {
    var s0, s1, s2, s3;

    s0 = peg$parseRegexp();
    if (s0 === peg$FAILED) {
      s0 = peg$parseGlob();
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        if (input.charCodeAt(peg$currPos) === 42) {
          s1 = peg$c90;
          peg$currPos++;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c91); }
        }
        if (s1 !== peg$FAILED) {
          s2 = peg$currPos;
          peg$silentFails++;
          s3 = peg$parseExprGuard();
          peg$silentFails--;
          if (s3 === peg$FAILED) {
            s2 = void 0;
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
          if (s2 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c243();
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          s1 = peg$parsePoolNameString();
          if (s1 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c244(s1);
          }
          s0 = s1;
        }
      }
    }

    return s0;
  }

  function peg$parsePoolNameString() {
    var s0;

    s0 = peg$parsePoolIdentifier();
    if (s0 === peg$FAILED) {
      s0 = peg$parseKSUID();
      if (s0 === peg$FAILED) {
        s0 = peg$parseQuotedString();
      }
    }

    return s0;
  }

  function peg$parsePoolIdentifier() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseIdentifierStart();
    if (s1 === peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 46) {
        s1 = peg$c116;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c117); }
      }
    }
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseIdentifierRest();
      if (s3 === peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 46) {
          s3 = peg$c116;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c117); }
        }
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseIdentifierRest();
        if (s3 === peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 46) {
            s3 = peg$c116;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c117); }
          }
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c82();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSortKeyArg() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 5) === peg$c245) {
        s2 = peg$c245;
        peg$currPos += 5;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c246); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseFieldExprs();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseOrderSuffix();
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c247(s4, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseTapArg() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 3) === peg$c248) {
        s2 = peg$c248;
        peg$currPos += 3;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c249); }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c250();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$c108;
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c251();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseFormatArg() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 6) === peg$c252) {
        s2 = peg$c252;
        peg$currPos += 6;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c253); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseIdentifierName();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c209(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseOrderSuffix() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c254) {
      s1 = peg$c254;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c255); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c256();
    }
    s0 = s1;
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 5) === peg$c257) {
        s1 = peg$c257;
        peg$currPos += 5;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c258); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c259();
      }
      s0 = s1;
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        s1 = peg$c108;
        if (s1 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c256();
        }
        s0 = s1;
      }
    }

    return s0;
  }

  function peg$parsePassOp() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c260) {
      s1 = peg$c260;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c261); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseEOKW();
      peg$silentFails--;
      if (s3 !== peg$FAILED) {
        peg$currPos = s2;
        s2 = void 0;
      } else {
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c262();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseExplodeOp() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 7) === peg$c263) {
      s1 = peg$c263;
      peg$currPos += 7;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c264); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseExprs();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseTypeArg();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseAsArg();
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c265(s3, s4, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseMergeOp() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5) === peg$c266) {
      s1 = peg$c266;
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c267); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseConditionalExpr();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c268(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseOverOp() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c269) {
      s1 = peg$c269;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c270); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseExprs();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseLocals();
          if (s4 === peg$FAILED) {
            s4 = null;
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parseLateral();
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c271(s3, s4, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseLateral() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 2) === peg$c50) {
        s2 = peg$c50;
        peg$currPos += 2;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c51); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 40) {
            s4 = peg$c16;
            peg$currPos++;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c17); }
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parse__();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseScope();
              if (s6 !== peg$FAILED) {
                s7 = peg$parse__();
                if (s7 !== peg$FAILED) {
                  if (input.charCodeAt(peg$currPos) === 41) {
                    s8 = peg$c18;
                    peg$currPos++;
                  } else {
                    s8 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c19); }
                  }
                  if (s8 !== peg$FAILED) {
                    peg$savedPos = s0;
                    s1 = peg$c43(s6);
                    s0 = s1;
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parse__();
      if (s1 !== peg$FAILED) {
        if (input.substr(peg$currPos, 2) === peg$c50) {
          s2 = peg$c50;
          peg$currPos += 2;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c51); }
        }
        if (s2 !== peg$FAILED) {
          s3 = peg$parse__();
          if (s3 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 40) {
              s4 = peg$c16;
              peg$currPos++;
            } else {
              s4 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c17); }
            }
            if (s4 !== peg$FAILED) {
              s5 = peg$parse__();
              if (s5 !== peg$FAILED) {
                s6 = peg$parseSeq();
                if (s6 !== peg$FAILED) {
                  s7 = peg$parse__();
                  if (s7 !== peg$FAILED) {
                    if (input.charCodeAt(peg$currPos) === 41) {
                      s8 = peg$c18;
                      peg$currPos++;
                    } else {
                      s8 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c19); }
                    }
                    if (s8 !== peg$FAILED) {
                      peg$savedPos = s0;
                      s1 = peg$c1(s6);
                      s0 = s1;
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseLocals() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 4) === peg$c103) {
        s2 = peg$c103;
        peg$currPos += 4;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c104); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseLocalsAssignment();
          if (s4 !== peg$FAILED) {
            s5 = [];
            s6 = peg$currPos;
            s7 = peg$parse__();
            if (s7 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 44) {
                s8 = peg$c26;
                peg$currPos++;
              } else {
                s8 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c27); }
              }
              if (s8 !== peg$FAILED) {
                s9 = peg$parse__();
                if (s9 !== peg$FAILED) {
                  s10 = peg$parseLocalsAssignment();
                  if (s10 !== peg$FAILED) {
                    peg$savedPos = s6;
                    s7 = peg$c272(s4, s10);
                    s6 = s7;
                  } else {
                    peg$currPos = s6;
                    s6 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s6;
                  s6 = peg$FAILED;
                }
              } else {
                peg$currPos = s6;
                s6 = peg$FAILED;
              }
            } else {
              peg$currPos = s6;
              s6 = peg$FAILED;
            }
            while (s6 !== peg$FAILED) {
              s5.push(s6);
              s6 = peg$currPos;
              s7 = peg$parse__();
              if (s7 !== peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 44) {
                  s8 = peg$c26;
                  peg$currPos++;
                } else {
                  s8 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c27); }
                }
                if (s8 !== peg$FAILED) {
                  s9 = peg$parse__();
                  if (s9 !== peg$FAILED) {
                    s10 = peg$parseLocalsAssignment();
                    if (s10 !== peg$FAILED) {
                      peg$savedPos = s6;
                      s7 = peg$c272(s4, s10);
                      s6 = s7;
                    } else {
                      peg$currPos = s6;
                      s6 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s6;
                    s6 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s6;
                  s6 = peg$FAILED;
                }
              } else {
                peg$currPos = s6;
                s6 = peg$FAILED;
              }
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c3(s4, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseLocalsAssignment() {
    var s0, s1, s2, s3, s4, s5, s6;

    s0 = peg$currPos;
    s1 = peg$parseIdentifierName();
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      s3 = peg$parse__();
      if (s3 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 61) {
          s4 = peg$c8;
          peg$currPos++;
        } else {
          s4 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c9); }
        }
        if (s4 !== peg$FAILED) {
          s5 = peg$parse__();
          if (s5 !== peg$FAILED) {
            s6 = peg$parseConditionalExpr();
            if (s6 !== peg$FAILED) {
              s3 = [s3, s4, s5, s6];
              s2 = s3;
            } else {
              peg$currPos = s2;
              s2 = peg$FAILED;
            }
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 === peg$FAILED) {
        s2 = null;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c273(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseYieldOp() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5) === peg$c274) {
      s1 = peg$c274;
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c275); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseExprs();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c276(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseTypeArg() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseBY();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseType();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c277(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseAsArg() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseAS();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseDerefExpr();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c278(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFieldExprs() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseDerefExpr();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s5 = peg$c26;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c27); }
        }
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseDerefExpr();
            if (s7 !== peg$FAILED) {
              s4 = [s4, s5, s6, s7];
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 44) {
            s5 = peg$c26;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c27); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseDerefExpr();
              if (s7 !== peg$FAILED) {
                s4 = [s4, s5, s6, s7];
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c280(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseAssignments() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseAssignment();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s5 = peg$c26;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c27); }
        }
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseAssignment();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c272(s1, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 44) {
            s5 = peg$c26;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c27); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseAssignment();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c272(s1, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c281(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseAssignment() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parseDerefExpr();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.substr(peg$currPos, 2) === peg$c112) {
          s3 = peg$c112;
          peg$currPos += 2;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c113); }
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseConditionalExpr();
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c282(s1, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseExpr() {
    var s0;

    s0 = peg$parseConditionalExpr();

    return s0;
  }

  function peg$parseConditionalExpr() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10;

    s0 = peg$currPos;
    s1 = peg$parseLogicalOrExpr();
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      s3 = peg$parse__();
      if (s3 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 63) {
          s4 = peg$c283;
          peg$currPos++;
        } else {
          s4 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c284); }
        }
        if (s4 !== peg$FAILED) {
          s5 = peg$parse__();
          if (s5 !== peg$FAILED) {
            s6 = peg$parseConditionalExpr();
            if (s6 !== peg$FAILED) {
              s7 = peg$parse__();
              if (s7 !== peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 58) {
                  s8 = peg$c20;
                  peg$currPos++;
                } else {
                  s8 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c21); }
                }
                if (s8 !== peg$FAILED) {
                  s9 = peg$parse__();
                  if (s9 !== peg$FAILED) {
                    s10 = peg$parseConditionalExpr();
                    if (s10 !== peg$FAILED) {
                      s3 = [s3, s4, s5, s6, s7, s8, s9, s10];
                      s2 = s3;
                    } else {
                      peg$currPos = s2;
                      s2 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s2;
                    s2 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s2;
                  s2 = peg$FAILED;
                }
              } else {
                peg$currPos = s2;
                s2 = peg$FAILED;
              }
            } else {
              peg$currPos = s2;
              s2 = peg$FAILED;
            }
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 === peg$FAILED) {
        s2 = null;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c285(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseLogicalOrExpr() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseLogicalAndExpr();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        s5 = peg$parseOrToken();
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseLogicalAndExpr();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c286(s1, s5, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          s5 = peg$parseOrToken();
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseLogicalAndExpr();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c286(s1, s5, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c287(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseLogicalAndExpr() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseComparisonExpr();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        s5 = peg$parseAndToken();
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseComparisonExpr();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c286(s1, s5, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          s5 = peg$parseAndToken();
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseComparisonExpr();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c286(s1, s5, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c287(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseComparisonExpr() {
    var s0, s1, s2, s3, s4, s5, s6;

    s0 = peg$currPos;
    s1 = peg$parseAdditiveExpr();
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      s3 = peg$parse__();
      if (s3 !== peg$FAILED) {
        s4 = peg$parseComparator();
        if (s4 !== peg$FAILED) {
          s5 = peg$parse__();
          if (s5 !== peg$FAILED) {
            s6 = peg$parseAdditiveExpr();
            if (s6 !== peg$FAILED) {
              s3 = [s3, s4, s5, s6];
              s2 = s3;
            } else {
              peg$currPos = s2;
              s2 = peg$FAILED;
            }
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 === peg$FAILED) {
        s2 = peg$currPos;
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$currPos;
          if (input.charCodeAt(peg$currPos) === 126) {
            s5 = peg$c66;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c67); }
          }
          if (s5 !== peg$FAILED) {
            peg$savedPos = s4;
            s5 = peg$c288();
          }
          s4 = s5;
          if (s4 !== peg$FAILED) {
            s5 = peg$parse__();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseRegexp();
              if (s6 !== peg$FAILED) {
                s3 = [s3, s4, s5, s6];
                s2 = s3;
              } else {
                peg$currPos = s2;
                s2 = peg$FAILED;
              }
            } else {
              peg$currPos = s2;
              s2 = peg$FAILED;
            }
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      }
      if (s2 === peg$FAILED) {
        s2 = null;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c289(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseAdditiveExpr() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseMultiplicativeExpr();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        s5 = peg$parseAdditiveOperator();
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseMultiplicativeExpr();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c286(s1, s5, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          s5 = peg$parseAdditiveOperator();
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseMultiplicativeExpr();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c286(s1, s5, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c287(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseAdditiveOperator() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 43) {
      s1 = peg$c290;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c291); }
    }
    if (s1 === peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 45) {
        s1 = peg$c292;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c293); }
      }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c82();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseMultiplicativeExpr() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseNotExpr();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        s5 = peg$parseMultiplicativeOperator();
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseNotExpr();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c286(s1, s5, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          s5 = peg$parseMultiplicativeOperator();
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseNotExpr();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c286(s1, s5, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c287(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseMultiplicativeOperator() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 42) {
      s1 = peg$c90;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c91); }
    }
    if (s1 === peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 47) {
        s1 = peg$c294;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c295); }
      }
      if (s1 === peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 37) {
          s1 = peg$c296;
          peg$currPos++;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c297); }
        }
      }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c82();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseNotExpr() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 33) {
      s1 = peg$c86;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c87); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseNotExpr();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c298(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$parseNegationExpr();
    }

    return s0;
  }

  function peg$parseNegationExpr() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$currPos;
    peg$silentFails++;
    s2 = peg$parseLiteral();
    peg$silentFails--;
    if (s2 === peg$FAILED) {
      s1 = void 0;
    } else {
      peg$currPos = s1;
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 45) {
        s2 = peg$c292;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c293); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseFuncExpr();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c299(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$parseFuncExpr();
    }

    return s0;
  }

  function peg$parseFuncExpr() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseCast();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseDeref();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseDeref();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c83(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseFunction();
      if (s1 !== peg$FAILED) {
        s2 = [];
        s3 = peg$parseDeref();
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          s3 = peg$parseDeref();
        }
        if (s2 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c83(s1, s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$parseDerefExpr();
        if (s0 === peg$FAILED) {
          s0 = peg$parsePrimary();
        }
      }
    }

    return s0;
  }

  function peg$parseFuncGuard() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseNotFuncs();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 40) {
          s3 = peg$c16;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c17); }
        }
        if (s3 !== peg$FAILED) {
          s1 = [s1, s2, s3];
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseNotFuncs() {
    var s0;

    if (input.substr(peg$currPos, 3) === peg$c300) {
      s0 = peg$c300;
      peg$currPos += 3;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c301); }
    }
    if (s0 === peg$FAILED) {
      if (input.substr(peg$currPos, 6) === peg$c302) {
        s0 = peg$c302;
        peg$currPos += 6;
      } else {
        s0 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c303); }
      }
    }

    return s0;
  }

  function peg$parseCast() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseCastType();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 40) {
          s3 = peg$c16;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c17); }
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseOverExpr();
            if (s5 === peg$FAILED) {
              s5 = peg$parseConditionalExpr();
            }
            if (s5 !== peg$FAILED) {
              s6 = peg$parse__();
              if (s6 !== peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 41) {
                  s7 = peg$c18;
                  peg$currPos++;
                } else {
                  s7 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c19); }
                }
                if (s7 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c304(s1, s5);
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFunction() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10, s11, s12, s13, s14, s15, s16;

    s0 = peg$parseGrep();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 6) === peg$c305) {
        s1 = peg$c305;
        peg$currPos += 6;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c306); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parse__();
        if (s2 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 40) {
            s3 = peg$c16;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c17); }
          }
          if (s3 !== peg$FAILED) {
            s4 = peg$parse__();
            if (s4 !== peg$FAILED) {
              s5 = peg$parseRegexpPattern();
              if (s5 !== peg$FAILED) {
                s6 = peg$parse__();
                if (s6 !== peg$FAILED) {
                  if (input.charCodeAt(peg$currPos) === 44) {
                    s7 = peg$c26;
                    peg$currPos++;
                  } else {
                    s7 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c27); }
                  }
                  if (s7 !== peg$FAILED) {
                    s8 = peg$parse__();
                    if (s8 !== peg$FAILED) {
                      s9 = peg$parseConditionalExpr();
                      if (s9 !== peg$FAILED) {
                        s10 = peg$parse__();
                        if (s10 !== peg$FAILED) {
                          if (input.charCodeAt(peg$currPos) === 41) {
                            s11 = peg$c18;
                            peg$currPos++;
                          } else {
                            s11 = peg$FAILED;
                            if (peg$silentFails === 0) { peg$fail(peg$c19); }
                          }
                          if (s11 !== peg$FAILED) {
                            s12 = peg$parseWhereClause();
                            if (s12 === peg$FAILED) {
                              s12 = null;
                            }
                            if (s12 !== peg$FAILED) {
                              peg$savedPos = s0;
                              s1 = peg$c307(s5, s9, s12);
                              s0 = s1;
                            } else {
                              peg$currPos = s0;
                              s0 = peg$FAILED;
                            }
                          } else {
                            peg$currPos = s0;
                            s0 = peg$FAILED;
                          }
                        } else {
                          peg$currPos = s0;
                          s0 = peg$FAILED;
                        }
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        if (input.substr(peg$currPos, 14) === peg$c308) {
          s1 = peg$c308;
          peg$currPos += 14;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c309); }
        }
        if (s1 !== peg$FAILED) {
          s2 = peg$parse__();
          if (s2 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 40) {
              s3 = peg$c16;
              peg$currPos++;
            } else {
              s3 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c17); }
            }
            if (s3 !== peg$FAILED) {
              s4 = peg$parse__();
              if (s4 !== peg$FAILED) {
                s5 = peg$parseConditionalExpr();
                if (s5 !== peg$FAILED) {
                  s6 = peg$parse__();
                  if (s6 !== peg$FAILED) {
                    if (input.charCodeAt(peg$currPos) === 44) {
                      s7 = peg$c26;
                      peg$currPos++;
                    } else {
                      s7 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c27); }
                    }
                    if (s7 !== peg$FAILED) {
                      s8 = peg$parse__();
                      if (s8 !== peg$FAILED) {
                        s9 = peg$parseRegexpPattern();
                        if (s9 !== peg$FAILED) {
                          s10 = peg$parse__();
                          if (s10 !== peg$FAILED) {
                            if (input.charCodeAt(peg$currPos) === 44) {
                              s11 = peg$c26;
                              peg$currPos++;
                            } else {
                              s11 = peg$FAILED;
                              if (peg$silentFails === 0) { peg$fail(peg$c27); }
                            }
                            if (s11 !== peg$FAILED) {
                              s12 = peg$parse__();
                              if (s12 !== peg$FAILED) {
                                s13 = peg$parseConditionalExpr();
                                if (s13 !== peg$FAILED) {
                                  s14 = peg$parse__();
                                  if (s14 !== peg$FAILED) {
                                    if (input.charCodeAt(peg$currPos) === 41) {
                                      s15 = peg$c18;
                                      peg$currPos++;
                                    } else {
                                      s15 = peg$FAILED;
                                      if (peg$silentFails === 0) { peg$fail(peg$c19); }
                                    }
                                    if (s15 !== peg$FAILED) {
                                      s16 = peg$parseWhereClause();
                                      if (s16 === peg$FAILED) {
                                        s16 = null;
                                      }
                                      if (s16 !== peg$FAILED) {
                                        peg$savedPos = s0;
                                        s1 = peg$c310(s5, s9, s13, s16);
                                        s0 = s1;
                                      } else {
                                        peg$currPos = s0;
                                        s0 = peg$FAILED;
                                      }
                                    } else {
                                      peg$currPos = s0;
                                      s0 = peg$FAILED;
                                    }
                                  } else {
                                    peg$currPos = s0;
                                    s0 = peg$FAILED;
                                  }
                                } else {
                                  peg$currPos = s0;
                                  s0 = peg$FAILED;
                                }
                              } else {
                                peg$currPos = s0;
                                s0 = peg$FAILED;
                              }
                            } else {
                              peg$currPos = s0;
                              s0 = peg$FAILED;
                            }
                          } else {
                            peg$currPos = s0;
                            s0 = peg$FAILED;
                          }
                        } else {
                          peg$currPos = s0;
                          s0 = peg$FAILED;
                        }
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          s1 = peg$currPos;
          peg$silentFails++;
          s2 = peg$parseFuncGuard();
          peg$silentFails--;
          if (s2 === peg$FAILED) {
            s1 = void 0;
          } else {
            peg$currPos = s1;
            s1 = peg$FAILED;
          }
          if (s1 !== peg$FAILED) {
            s2 = peg$parseIdentifierName();
            if (s2 !== peg$FAILED) {
              s3 = peg$parse__();
              if (s3 !== peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 40) {
                  s4 = peg$c16;
                  peg$currPos++;
                } else {
                  s4 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c17); }
                }
                if (s4 !== peg$FAILED) {
                  s5 = peg$parse__();
                  if (s5 !== peg$FAILED) {
                    s6 = peg$parseFunctionArgs();
                    if (s6 !== peg$FAILED) {
                      s7 = peg$parse__();
                      if (s7 !== peg$FAILED) {
                        if (input.charCodeAt(peg$currPos) === 41) {
                          s8 = peg$c18;
                          peg$currPos++;
                        } else {
                          s8 = peg$FAILED;
                          if (peg$silentFails === 0) { peg$fail(peg$c19); }
                        }
                        if (s8 !== peg$FAILED) {
                          s9 = peg$parseWhereClause();
                          if (s9 === peg$FAILED) {
                            s9 = null;
                          }
                          if (s9 !== peg$FAILED) {
                            peg$savedPos = s0;
                            s1 = peg$c311(s2, s6, s9);
                            s0 = s1;
                          } else {
                            peg$currPos = s0;
                            s0 = peg$FAILED;
                          }
                        } else {
                          peg$currPos = s0;
                          s0 = peg$FAILED;
                        }
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        }
      }
    }

    return s0;
  }

  function peg$parseFunctionArgs() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseOverExpr();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c312(s1);
    }
    s0 = s1;
    if (s0 === peg$FAILED) {
      s0 = peg$parseOptionalExprs();
    }

    return s0;
  }

  function peg$parseGrep() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10, s11;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c313) {
      s1 = peg$c313;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c314); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 40) {
          s3 = peg$c16;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c17); }
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            s5 = peg$parsePattern();
            if (s5 !== peg$FAILED) {
              s6 = peg$parse__();
              if (s6 !== peg$FAILED) {
                s7 = peg$currPos;
                if (input.charCodeAt(peg$currPos) === 44) {
                  s8 = peg$c26;
                  peg$currPos++;
                } else {
                  s8 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c27); }
                }
                if (s8 !== peg$FAILED) {
                  s9 = peg$parse__();
                  if (s9 !== peg$FAILED) {
                    s10 = peg$parseOverExpr();
                    if (s10 === peg$FAILED) {
                      s10 = peg$parseConditionalExpr();
                    }
                    if (s10 !== peg$FAILED) {
                      s11 = peg$parse__();
                      if (s11 !== peg$FAILED) {
                        s8 = [s8, s9, s10, s11];
                        s7 = s8;
                      } else {
                        peg$currPos = s7;
                        s7 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s7;
                      s7 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s7;
                    s7 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s7;
                  s7 = peg$FAILED;
                }
                if (s7 === peg$FAILED) {
                  s7 = null;
                }
                if (s7 !== peg$FAILED) {
                  if (input.charCodeAt(peg$currPos) === 41) {
                    s8 = peg$c18;
                    peg$currPos++;
                  } else {
                    s8 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c19); }
                  }
                  if (s8 !== peg$FAILED) {
                    peg$savedPos = s0;
                    s1 = peg$c315(s5, s7);
                    s0 = s1;
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePattern() {
    var s0, s1;

    s0 = peg$parseRegexp();
    if (s0 === peg$FAILED) {
      s0 = peg$parseGlob();
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        s1 = peg$parseQuotedString();
        if (s1 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c316(s1);
        }
        s0 = s1;
      }
    }

    return s0;
  }

  function peg$parseOptionalExprs() {
    var s0, s1;

    s0 = peg$parseExprs();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parse__();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c317();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseExprs() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseConditionalExpr();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s5 = peg$c26;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c27); }
        }
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseConditionalExpr();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c318(s1, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 44) {
            s5 = peg$c26;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c27); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseConditionalExpr();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c318(s1, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c3(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseDerefExpr() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$currPos;
    peg$silentFails++;
    s2 = peg$parseIP6();
    peg$silentFails--;
    if (s2 === peg$FAILED) {
      s1 = void 0;
    } else {
      peg$currPos = s1;
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseIdentifier();
      if (s2 !== peg$FAILED) {
        s3 = [];
        s4 = peg$parseDeref();
        while (s4 !== peg$FAILED) {
          s3.push(s4);
          s4 = peg$parseDeref();
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c83(s2, s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseDeref() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 91) {
      s1 = peg$c56;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c57); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseAdditiveExpr();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 58) {
            s4 = peg$c20;
            peg$currPos++;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c21); }
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parse__();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseAdditiveExpr();
              if (s6 === peg$FAILED) {
                s6 = null;
              }
              if (s6 !== peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 93) {
                  s7 = peg$c319;
                  peg$currPos++;
                } else {
                  s7 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c320); }
                }
                if (s7 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c321(s2, s6);
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 91) {
        s1 = peg$c56;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c57); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parse__();
        if (s2 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 58) {
            s3 = peg$c20;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c21); }
          }
          if (s3 !== peg$FAILED) {
            s4 = peg$parse__();
            if (s4 !== peg$FAILED) {
              s5 = peg$parseAdditiveExpr();
              if (s5 !== peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 93) {
                  s6 = peg$c319;
                  peg$currPos++;
                } else {
                  s6 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c320); }
                }
                if (s6 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c322(s5);
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        if (input.charCodeAt(peg$currPos) === 91) {
          s1 = peg$c56;
          peg$currPos++;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c57); }
        }
        if (s1 !== peg$FAILED) {
          s2 = peg$parseConditionalExpr();
          if (s2 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 93) {
              s3 = peg$c319;
              peg$currPos++;
            } else {
              s3 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c320); }
            }
            if (s3 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c323(s2);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          if (input.charCodeAt(peg$currPos) === 46) {
            s1 = peg$c116;
            peg$currPos++;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c117); }
          }
          if (s1 !== peg$FAILED) {
            s2 = peg$parseIdentifier();
            if (s2 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c324(s2);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        }
      }
    }

    return s0;
  }

  function peg$parsePrimary() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$parseRecord();
    if (s0 === peg$FAILED) {
      s0 = peg$parseArray();
      if (s0 === peg$FAILED) {
        s0 = peg$parseSet();
        if (s0 === peg$FAILED) {
          s0 = peg$parseMap();
          if (s0 === peg$FAILED) {
            s0 = peg$parseLiteral();
            if (s0 === peg$FAILED) {
              s0 = peg$currPos;
              if (input.charCodeAt(peg$currPos) === 40) {
                s1 = peg$c16;
                peg$currPos++;
              } else {
                s1 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c17); }
              }
              if (s1 !== peg$FAILED) {
                s2 = peg$parse__();
                if (s2 !== peg$FAILED) {
                  s3 = peg$parseOverExpr();
                  if (s3 !== peg$FAILED) {
                    s4 = peg$parse__();
                    if (s4 !== peg$FAILED) {
                      if (input.charCodeAt(peg$currPos) === 41) {
                        s5 = peg$c18;
                        peg$currPos++;
                      } else {
                        s5 = peg$FAILED;
                        if (peg$silentFails === 0) { peg$fail(peg$c19); }
                      }
                      if (s5 !== peg$FAILED) {
                        peg$savedPos = s0;
                        s1 = peg$c61(s3);
                        s0 = s1;
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
              if (s0 === peg$FAILED) {
                s0 = peg$currPos;
                if (input.charCodeAt(peg$currPos) === 40) {
                  s1 = peg$c16;
                  peg$currPos++;
                } else {
                  s1 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c17); }
                }
                if (s1 !== peg$FAILED) {
                  s2 = peg$parse__();
                  if (s2 !== peg$FAILED) {
                    s3 = peg$parseConditionalExpr();
                    if (s3 !== peg$FAILED) {
                      s4 = peg$parse__();
                      if (s4 !== peg$FAILED) {
                        if (input.charCodeAt(peg$currPos) === 41) {
                          s5 = peg$c18;
                          peg$currPos++;
                        } else {
                          s5 = peg$FAILED;
                          if (peg$silentFails === 0) { peg$fail(peg$c19); }
                        }
                        if (s5 !== peg$FAILED) {
                          peg$savedPos = s0;
                          s1 = peg$c61(s3);
                          s0 = s1;
                        } else {
                          peg$currPos = s0;
                          s0 = peg$FAILED;
                        }
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              }
            }
          }
        }
      }
    }

    return s0;
  }

  function peg$parseOverExpr() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c269) {
      s1 = peg$c269;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c270); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseExprs();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseLocals();
          if (s4 === peg$FAILED) {
            s4 = null;
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parse__();
            if (s5 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 124) {
                s6 = peg$c52;
                peg$currPos++;
              } else {
                s6 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c53); }
              }
              if (s6 !== peg$FAILED) {
                s7 = peg$parse__();
                if (s7 !== peg$FAILED) {
                  s8 = peg$parseSeq();
                  if (s8 !== peg$FAILED) {
                    peg$savedPos = s0;
                    s1 = peg$c325(s3, s4, s8);
                    s0 = s1;
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseRecord() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 123) {
      s1 = peg$c54;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c55); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseRecordElems();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 125) {
              s5 = peg$c326;
              peg$currPos++;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c327); }
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c328(s3);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseRecordElems() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseRecordElem();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseRecordElemTail();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseRecordElemTail();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c281(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parse__();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c317();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseRecordElemTail() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 44) {
        s2 = peg$c26;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c27); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseRecordElem();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c329(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseRecordElem() {
    var s0;

    s0 = peg$parseSpread();
    if (s0 === peg$FAILED) {
      s0 = peg$parseField();
      if (s0 === peg$FAILED) {
        s0 = peg$parseIdentifier();
      }
    }

    return s0;
  }

  function peg$parseSpread() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 3) === peg$c29) {
      s1 = peg$c29;
      peg$currPos += 3;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c30); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseConditionalExpr();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c330(s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseField() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parseFieldName();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 58) {
          s3 = peg$c20;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c21); }
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseConditionalExpr();
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c331(s1, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseArray() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 91) {
      s1 = peg$c56;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c57); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseVectorElems();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 93) {
              s5 = peg$c319;
              peg$currPos++;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c320); }
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c332(s3);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSet() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c333) {
      s1 = peg$c333;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c334); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseVectorElems();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            if (input.substr(peg$currPos, 2) === peg$c335) {
              s5 = peg$c335;
              peg$currPos += 2;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c336); }
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c337(s3);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseVectorElems() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseVectorElem();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s5 = peg$c26;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c27); }
        }
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseVectorElem();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c318(s1, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 44) {
            s5 = peg$c26;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c27); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseVectorElem();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c318(s1, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c3(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parse__();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c317();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseVectorElem() {
    var s0, s1;

    s0 = peg$parseSpread();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseConditionalExpr();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c338(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseMap() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c339) {
      s1 = peg$c339;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c340); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseEntries();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            if (input.substr(peg$currPos, 2) === peg$c341) {
              s5 = peg$c341;
              peg$currPos += 2;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c342); }
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c343(s3);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseEntries() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseEntry();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseEntryTail();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseEntryTail();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c281(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parse__();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c317();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseEntryTail() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 44) {
        s2 = peg$c26;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c27); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseEntry();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c344(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseEntry() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parseConditionalExpr();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 58) {
          s3 = peg$c20;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c21); }
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseConditionalExpr();
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c345(s1, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLOp() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8;

    s0 = peg$currPos;
    s1 = peg$parseSQLSelect();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseSQLFrom();
      if (s2 === peg$FAILED) {
        s2 = null;
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseSQLJoins();
        if (s3 === peg$FAILED) {
          s3 = null;
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parseSQLWhere();
          if (s4 === peg$FAILED) {
            s4 = null;
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parseSQLGroupBy();
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              s6 = peg$parseSQLHaving();
              if (s6 === peg$FAILED) {
                s6 = null;
              }
              if (s6 !== peg$FAILED) {
                s7 = peg$parseSQLOrderBy();
                if (s7 === peg$FAILED) {
                  s7 = null;
                }
                if (s7 !== peg$FAILED) {
                  s8 = peg$parseSQLLimit();
                  if (s8 !== peg$FAILED) {
                    peg$savedPos = s0;
                    s1 = peg$c346(s1, s2, s3, s4, s5, s6, s7, s8);
                    s0 = s1;
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLSelect() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseSELECT();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 42) {
          s3 = peg$c90;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c91); }
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c64();
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseSELECT();
      if (s1 !== peg$FAILED) {
        s2 = peg$parse_();
        if (s2 !== peg$FAILED) {
          s3 = peg$parseSQLAssignments();
          if (s3 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c347(s3);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseSQLAssignment() {
    var s0, s1, s2, s3, s4, s5, s6;

    s0 = peg$currPos;
    s1 = peg$parseConditionalExpr();
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      s3 = peg$parse_();
      if (s3 !== peg$FAILED) {
        s4 = peg$parseAS();
        if (s4 !== peg$FAILED) {
          s5 = peg$parse_();
          if (s5 !== peg$FAILED) {
            s6 = peg$parseDerefExpr();
            if (s6 !== peg$FAILED) {
              s3 = [s3, s4, s5, s6];
              s2 = s3;
            } else {
              peg$currPos = s2;
              s2 = peg$FAILED;
            }
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 === peg$FAILED) {
        s2 = null;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c348(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLAssignments() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseSQLAssignment();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s5 = peg$c26;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c27); }
        }
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseSQLAssignment();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c111(s1, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 44) {
            s5 = peg$c26;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c27); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseSQLAssignment();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c111(s1, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c3(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLFrom() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseFROM();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseConditionalExpr();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseSQLAlias();
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c349(s4, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parse_();
      if (s1 !== peg$FAILED) {
        s2 = peg$parseFROM();
        if (s2 !== peg$FAILED) {
          s3 = peg$parse_();
          if (s3 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 42) {
              s4 = peg$c90;
              peg$currPos++;
            } else {
              s4 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c91); }
            }
            if (s4 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c64();
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseSQLAlias() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseAS();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseDerefExpr();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c236(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parse_();
      if (s1 !== peg$FAILED) {
        s2 = peg$currPos;
        peg$silentFails++;
        s3 = peg$currPos;
        s4 = peg$parseSQLTokenSentinels();
        if (s4 !== peg$FAILED) {
          s5 = peg$parse_();
          if (s5 !== peg$FAILED) {
            s4 = [s4, s5];
            s3 = s4;
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
        peg$silentFails--;
        if (s3 === peg$FAILED) {
          s2 = void 0;
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
        if (s2 !== peg$FAILED) {
          s3 = peg$parseDerefExpr();
          if (s3 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c236(s3);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseSQLJoins() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parseSQLJoin();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parseSQLJoin();
      if (s4 !== peg$FAILED) {
        peg$savedPos = s3;
        s4 = peg$c350(s1, s4);
      }
      s3 = s4;
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parseSQLJoin();
        if (s4 !== peg$FAILED) {
          peg$savedPos = s3;
          s4 = peg$c350(s1, s4);
        }
        s3 = s4;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c3(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLJoin() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9, s10, s11, s12, s13, s14;

    s0 = peg$currPos;
    s1 = peg$parseSQLJoinStyle();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse_();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseJOIN();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse_();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseConditionalExpr();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseSQLAlias();
              if (s6 === peg$FAILED) {
                s6 = null;
              }
              if (s6 !== peg$FAILED) {
                s7 = peg$parse_();
                if (s7 !== peg$FAILED) {
                  s8 = peg$parseON();
                  if (s8 !== peg$FAILED) {
                    s9 = peg$parse_();
                    if (s9 !== peg$FAILED) {
                      s10 = peg$parseJoinKey();
                      if (s10 !== peg$FAILED) {
                        s11 = peg$parse__();
                        if (s11 !== peg$FAILED) {
                          if (input.charCodeAt(peg$currPos) === 61) {
                            s12 = peg$c8;
                            peg$currPos++;
                          } else {
                            s12 = peg$FAILED;
                            if (peg$silentFails === 0) { peg$fail(peg$c9); }
                          }
                          if (s12 !== peg$FAILED) {
                            s13 = peg$parse__();
                            if (s13 !== peg$FAILED) {
                              s14 = peg$parseJoinKey();
                              if (s14 !== peg$FAILED) {
                                peg$savedPos = s0;
                                s1 = peg$c351(s1, s5, s6, s10, s14);
                                s0 = s1;
                              } else {
                                peg$currPos = s0;
                                s0 = peg$FAILED;
                              }
                            } else {
                              peg$currPos = s0;
                              s0 = peg$FAILED;
                            }
                          } else {
                            peg$currPos = s0;
                            s0 = peg$FAILED;
                          }
                        } else {
                          peg$currPos = s0;
                          s0 = peg$FAILED;
                        }
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLJoinStyle() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseANTI();
      if (s2 === peg$FAILED) {
        s2 = peg$parseINNER();
        if (s2 === peg$FAILED) {
          s2 = peg$parseLEFT();
          if (s2 === peg$FAILED) {
            s2 = peg$parseRIGHT();
          }
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c352(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$c108;
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c190();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseSQLWhere() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseWHERE();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseLogicalOrExpr();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c61(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLGroupBy() {
    var s0, s1, s2, s3, s4, s5, s6;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseGROUP();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseBY();
          if (s4 !== peg$FAILED) {
            s5 = peg$parse_();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseFieldExprs();
              if (s6 !== peg$FAILED) {
                peg$savedPos = s0;
                s1 = peg$c102(s6);
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLHaving() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseHAVING();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseLogicalOrExpr();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c61(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLOrderBy() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseORDER();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseBY();
          if (s4 !== peg$FAILED) {
            s5 = peg$parse_();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseExprs();
              if (s6 !== peg$FAILED) {
                s7 = peg$parseSQLOrder();
                if (s7 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c353(s6, s7);
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseSQLOrder() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseASC();
      if (s2 === peg$FAILED) {
        s2 = peg$parseDESC();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c354(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$c108;
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c256();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseSQLLimit() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse_();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseLIMIT();
      if (s2 !== peg$FAILED) {
        s3 = peg$parse_();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseUInt();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c355(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$c108;
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c109();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseSELECT() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 6).toLowerCase() === peg$c302) {
      s1 = input.substr(peg$currPos, 6);
      peg$currPos += 6;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c356); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c357();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseAS() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2).toLowerCase() === peg$c358) {
      s1 = input.substr(peg$currPos, 2);
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c359); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c360();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseFROM() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4).toLowerCase() === peg$c40) {
      s1 = input.substr(peg$currPos, 4);
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c361); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c362();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseJOIN() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4).toLowerCase() === peg$c182) {
      s1 = input.substr(peg$currPos, 4);
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c363); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c364();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseWHERE() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5).toLowerCase() === peg$c119) {
      s1 = input.substr(peg$currPos, 5);
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c365); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c366();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseGROUP() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5).toLowerCase() === peg$c367) {
      s1 = input.substr(peg$currPos, 5);
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c368); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c369();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseBY() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2).toLowerCase() === peg$c370) {
      s1 = input.substr(peg$currPos, 2);
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c371); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c372();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseHAVING() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 6).toLowerCase() === peg$c373) {
      s1 = input.substr(peg$currPos, 6);
      peg$currPos += 6;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c374); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c375();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseORDER() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5).toLowerCase() === peg$c245) {
      s1 = input.substr(peg$currPos, 5);
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c376); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c377();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseON() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2).toLowerCase() === peg$c378) {
      s1 = input.substr(peg$currPos, 2);
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c379); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c380();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseLIMIT() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5).toLowerCase() === peg$c381) {
      s1 = input.substr(peg$currPos, 5);
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c382); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c383();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseASC() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 3).toLowerCase() === peg$c384) {
      s1 = input.substr(peg$currPos, 3);
      peg$currPos += 3;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c385); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c256();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseDESC() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4).toLowerCase() === peg$c386) {
      s1 = input.substr(peg$currPos, 4);
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c387); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c259();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseANTI() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4).toLowerCase() === peg$c185) {
      s1 = input.substr(peg$currPos, 4);
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c388); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c187();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseLEFT() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4).toLowerCase() === peg$c191) {
      s1 = input.substr(peg$currPos, 4);
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c389); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c193();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseRIGHT() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5).toLowerCase() === peg$c194) {
      s1 = input.substr(peg$currPos, 5);
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c390); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c196();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseINNER() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5).toLowerCase() === peg$c188) {
      s1 = input.substr(peg$currPos, 5);
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c391); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c190();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseSQLTokenSentinels() {
    var s0;

    s0 = peg$parseSELECT();
    if (s0 === peg$FAILED) {
      s0 = peg$parseAS();
      if (s0 === peg$FAILED) {
        s0 = peg$parseFROM();
        if (s0 === peg$FAILED) {
          s0 = peg$parseJOIN();
          if (s0 === peg$FAILED) {
            s0 = peg$parseWHERE();
            if (s0 === peg$FAILED) {
              s0 = peg$parseGROUP();
              if (s0 === peg$FAILED) {
                s0 = peg$parseHAVING();
                if (s0 === peg$FAILED) {
                  s0 = peg$parseORDER();
                  if (s0 === peg$FAILED) {
                    s0 = peg$parseLIMIT();
                    if (s0 === peg$FAILED) {
                      s0 = peg$parseON();
                    }
                  }
                }
              }
            }
          }
        }
      }
    }

    return s0;
  }

  function peg$parseLiteral() {
    var s0;

    s0 = peg$parseTypeLiteral();
    if (s0 === peg$FAILED) {
      s0 = peg$parseTemplateLiteral();
      if (s0 === peg$FAILED) {
        s0 = peg$parseSubnetLiteral();
        if (s0 === peg$FAILED) {
          s0 = peg$parseAddressLiteral();
          if (s0 === peg$FAILED) {
            s0 = peg$parseBytesLiteral();
            if (s0 === peg$FAILED) {
              s0 = peg$parseDuration();
              if (s0 === peg$FAILED) {
                s0 = peg$parseTime();
                if (s0 === peg$FAILED) {
                  s0 = peg$parseFloatLiteral();
                  if (s0 === peg$FAILED) {
                    s0 = peg$parseIntegerLiteral();
                    if (s0 === peg$FAILED) {
                      s0 = peg$parseBooleanLiteral();
                      if (s0 === peg$FAILED) {
                        s0 = peg$parseNullLiteral();
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }

    return s0;
  }

  function peg$parseSubnetLiteral() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseIP6Net();
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseIdentifierRest();
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c392(s1);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseIP4Net();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c392(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseAddressLiteral() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseIP6();
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseIdentifierRest();
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c393(s1);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseIP();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c393(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseFloatLiteral() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseFloatString();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c394(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseIntegerLiteral() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseIntString();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c395(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseBooleanLiteral() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c396) {
      s1 = peg$c396;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c397); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c398();
    }
    s0 = s1;
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.substr(peg$currPos, 5) === peg$c399) {
        s1 = peg$c399;
        peg$currPos += 5;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c400); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c401();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseNullLiteral() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 4) === peg$c402) {
      s1 = peg$c402;
      peg$currPos += 4;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c403); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c404();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseBytesLiteral() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c405) {
      s1 = peg$c405;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c406); }
    }
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseHexDigit();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseHexDigit();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c407();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseTypeLiteral() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 60) {
      s1 = peg$c76;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c77); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseType();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 62) {
          s3 = peg$c80;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c81); }
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c408(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseCastType() {
    var s0, s1;

    s0 = peg$parseTypeLiteral();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parsePrimitiveType();
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c408(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseType() {
    var s0;

    s0 = peg$parseTypeLiteral();
    if (s0 === peg$FAILED) {
      s0 = peg$parseAmbiguousType();
      if (s0 === peg$FAILED) {
        s0 = peg$parseComplexType();
      }
    }

    return s0;
  }

  function peg$parseAmbiguousType() {
    var s0, s1, s2, s3, s4, s5, s6;

    s0 = peg$currPos;
    s1 = peg$parsePrimitiveType();
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseIdentifierRest();
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c409(s1);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseIdentifierName();
      if (s1 !== peg$FAILED) {
        s2 = peg$currPos;
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 61) {
            s4 = peg$c8;
            peg$currPos++;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c9); }
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parse__();
            if (s5 !== peg$FAILED) {
              s6 = peg$parseType();
              if (s6 !== peg$FAILED) {
                s3 = [s3, s4, s5, s6];
                s2 = s3;
              } else {
                peg$currPos = s2;
                s2 = peg$FAILED;
              }
            } else {
              peg$currPos = s2;
              s2 = peg$FAILED;
            }
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
        if (s2 === peg$FAILED) {
          s2 = null;
        }
        if (s2 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c410(s1, s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        s1 = peg$parseQuotedString();
        if (s1 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c411(s1);
        }
        s0 = s1;
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          if (input.charCodeAt(peg$currPos) === 40) {
            s1 = peg$c16;
            peg$currPos++;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c17); }
          }
          if (s1 !== peg$FAILED) {
            s2 = peg$parse__();
            if (s2 !== peg$FAILED) {
              s3 = peg$parseTypeUnion();
              if (s3 !== peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 41) {
                  s4 = peg$c18;
                  peg$currPos++;
                } else {
                  s4 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c19); }
                }
                if (s4 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c412(s3);
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        }
      }
    }

    return s0;
  }

  function peg$parseTypeUnion() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseTypeList();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c413(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseTypeList() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseType();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseTypeListTail();
      if (s3 !== peg$FAILED) {
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          s3 = peg$parseTypeListTail();
        }
      } else {
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c281(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseTypeListTail() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 44) {
        s2 = peg$c26;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c27); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseType();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c277(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseComplexType() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 123) {
      s1 = peg$c54;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c55); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseTypeFieldList();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 125) {
              s5 = peg$c326;
              peg$currPos++;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c327); }
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c414(s3);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 91) {
        s1 = peg$c56;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c57); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parse__();
        if (s2 !== peg$FAILED) {
          s3 = peg$parseType();
          if (s3 !== peg$FAILED) {
            s4 = peg$parse__();
            if (s4 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 93) {
                s5 = peg$c319;
                peg$currPos++;
              } else {
                s5 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c320); }
              }
              if (s5 !== peg$FAILED) {
                peg$savedPos = s0;
                s1 = peg$c415(s3);
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        if (input.substr(peg$currPos, 2) === peg$c333) {
          s1 = peg$c333;
          peg$currPos += 2;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c334); }
        }
        if (s1 !== peg$FAILED) {
          s2 = peg$parse__();
          if (s2 !== peg$FAILED) {
            s3 = peg$parseType();
            if (s3 !== peg$FAILED) {
              s4 = peg$parse__();
              if (s4 !== peg$FAILED) {
                if (input.substr(peg$currPos, 2) === peg$c335) {
                  s5 = peg$c335;
                  peg$currPos += 2;
                } else {
                  s5 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c336); }
                }
                if (s5 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c416(s3);
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          if (input.substr(peg$currPos, 2) === peg$c339) {
            s1 = peg$c339;
            peg$currPos += 2;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c340); }
          }
          if (s1 !== peg$FAILED) {
            s2 = peg$parse__();
            if (s2 !== peg$FAILED) {
              s3 = peg$parseType();
              if (s3 !== peg$FAILED) {
                s4 = peg$parse__();
                if (s4 !== peg$FAILED) {
                  if (input.charCodeAt(peg$currPos) === 58) {
                    s5 = peg$c20;
                    peg$currPos++;
                  } else {
                    s5 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c21); }
                  }
                  if (s5 !== peg$FAILED) {
                    s6 = peg$parse__();
                    if (s6 !== peg$FAILED) {
                      s7 = peg$parseType();
                      if (s7 !== peg$FAILED) {
                        s8 = peg$parse__();
                        if (s8 !== peg$FAILED) {
                          if (input.substr(peg$currPos, 2) === peg$c341) {
                            s9 = peg$c341;
                            peg$currPos += 2;
                          } else {
                            s9 = peg$FAILED;
                            if (peg$silentFails === 0) { peg$fail(peg$c342); }
                          }
                          if (s9 !== peg$FAILED) {
                            peg$savedPos = s0;
                            s1 = peg$c417(s3, s7);
                            s0 = s1;
                          } else {
                            peg$currPos = s0;
                            s0 = peg$FAILED;
                          }
                        } else {
                          peg$currPos = s0;
                          s0 = peg$FAILED;
                        }
                      } else {
                        peg$currPos = s0;
                        s0 = peg$FAILED;
                      }
                    } else {
                      peg$currPos = s0;
                      s0 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s0;
                    s0 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        }
      }
    }

    return s0;
  }

  function peg$parseTemplateLiteral() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseTemplateLiteralParts();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c418(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseTemplateLiteralParts() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 34) {
      s1 = peg$c419;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c420); }
    }
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseTemplateDoubleQuotedPart();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseTemplateDoubleQuotedPart();
      }
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 34) {
          s3 = peg$c419;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c420); }
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c5(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 39) {
        s1 = peg$c421;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c422); }
      }
      if (s1 !== peg$FAILED) {
        s2 = [];
        s3 = peg$parseTemplateSingleQuotedPart();
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          s3 = peg$parseTemplateSingleQuotedPart();
        }
        if (s2 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 39) {
            s3 = peg$c421;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c422); }
          }
          if (s3 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c5(s2);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseTemplateDoubleQuotedPart() {
    var s0, s1, s2;

    s0 = peg$parseTemplateExpr();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parseTemplateDoubleQuotedChar();
      if (s2 !== peg$FAILED) {
        while (s2 !== peg$FAILED) {
          s1.push(s2);
          s2 = peg$parseTemplateDoubleQuotedChar();
        }
      } else {
        s1 = peg$FAILED;
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c423(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseTemplateDoubleQuotedChar() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 92) {
      s1 = peg$c424;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c425); }
    }
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 2) === peg$c426) {
        s2 = peg$c426;
        peg$currPos += 2;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c427); }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c5(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$currPos;
      peg$silentFails++;
      if (input.substr(peg$currPos, 2) === peg$c426) {
        s2 = peg$c426;
        peg$currPos += 2;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c427); }
      }
      peg$silentFails--;
      if (s2 === peg$FAILED) {
        s1 = void 0;
      } else {
        peg$currPos = s1;
        s1 = peg$FAILED;
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseDoubleQuotedChar();
        if (s2 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c5(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseTemplateSingleQuotedPart() {
    var s0, s1, s2;

    s0 = peg$parseTemplateExpr();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = [];
      s2 = peg$parseTemplateSingleQuotedChar();
      if (s2 !== peg$FAILED) {
        while (s2 !== peg$FAILED) {
          s1.push(s2);
          s2 = peg$parseTemplateSingleQuotedChar();
        }
      } else {
        s1 = peg$FAILED;
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c423(s1);
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseTemplateSingleQuotedChar() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 92) {
      s1 = peg$c424;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c425); }
    }
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 2) === peg$c426) {
        s2 = peg$c426;
        peg$currPos += 2;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c427); }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c5(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$currPos;
      peg$silentFails++;
      if (input.substr(peg$currPos, 2) === peg$c426) {
        s2 = peg$c426;
        peg$currPos += 2;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c427); }
      }
      peg$silentFails--;
      if (s2 === peg$FAILED) {
        s1 = void 0;
      } else {
        peg$currPos = s1;
        s1 = peg$FAILED;
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseSingleQuotedChar();
        if (s2 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c5(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseTemplateExpr() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c426) {
      s1 = peg$c426;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c427); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        s3 = peg$parseConditionalExpr();
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 125) {
              s5 = peg$c326;
              peg$currPos++;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c327); }
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c428(s3);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePrimitiveType() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 5) === peg$c429) {
      s1 = peg$c429;
      peg$currPos += 5;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c430); }
    }
    if (s1 === peg$FAILED) {
      if (input.substr(peg$currPos, 6) === peg$c431) {
        s1 = peg$c431;
        peg$currPos += 6;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c432); }
      }
      if (s1 === peg$FAILED) {
        if (input.substr(peg$currPos, 6) === peg$c433) {
          s1 = peg$c433;
          peg$currPos += 6;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c434); }
        }
        if (s1 === peg$FAILED) {
          if (input.substr(peg$currPos, 6) === peg$c435) {
            s1 = peg$c435;
            peg$currPos += 6;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c436); }
          }
          if (s1 === peg$FAILED) {
            if (input.substr(peg$currPos, 4) === peg$c437) {
              s1 = peg$c437;
              peg$currPos += 4;
            } else {
              s1 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c438); }
            }
            if (s1 === peg$FAILED) {
              if (input.substr(peg$currPos, 5) === peg$c439) {
                s1 = peg$c439;
                peg$currPos += 5;
              } else {
                s1 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c440); }
              }
              if (s1 === peg$FAILED) {
                if (input.substr(peg$currPos, 5) === peg$c441) {
                  s1 = peg$c441;
                  peg$currPos += 5;
                } else {
                  s1 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c442); }
                }
                if (s1 === peg$FAILED) {
                  if (input.substr(peg$currPos, 5) === peg$c443) {
                    s1 = peg$c443;
                    peg$currPos += 5;
                  } else {
                    s1 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c444); }
                  }
                  if (s1 === peg$FAILED) {
                    if (input.substr(peg$currPos, 7) === peg$c445) {
                      s1 = peg$c445;
                      peg$currPos += 7;
                    } else {
                      s1 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c446); }
                    }
                    if (s1 === peg$FAILED) {
                      if (input.substr(peg$currPos, 7) === peg$c447) {
                        s1 = peg$c447;
                        peg$currPos += 7;
                      } else {
                        s1 = peg$FAILED;
                        if (peg$silentFails === 0) { peg$fail(peg$c448); }
                      }
                      if (s1 === peg$FAILED) {
                        if (input.substr(peg$currPos, 7) === peg$c449) {
                          s1 = peg$c449;
                          peg$currPos += 7;
                        } else {
                          s1 = peg$FAILED;
                          if (peg$silentFails === 0) { peg$fail(peg$c450); }
                        }
                        if (s1 === peg$FAILED) {
                          if (input.substr(peg$currPos, 4) === peg$c451) {
                            s1 = peg$c451;
                            peg$currPos += 4;
                          } else {
                            s1 = peg$FAILED;
                            if (peg$silentFails === 0) { peg$fail(peg$c452); }
                          }
                          if (s1 === peg$FAILED) {
                            if (input.substr(peg$currPos, 6) === peg$c453) {
                              s1 = peg$c453;
                              peg$currPos += 6;
                            } else {
                              s1 = peg$FAILED;
                              if (peg$silentFails === 0) { peg$fail(peg$c454); }
                            }
                            if (s1 === peg$FAILED) {
                              if (input.substr(peg$currPos, 8) === peg$c455) {
                                s1 = peg$c455;
                                peg$currPos += 8;
                              } else {
                                s1 = peg$FAILED;
                                if (peg$silentFails === 0) { peg$fail(peg$c456); }
                              }
                              if (s1 === peg$FAILED) {
                                if (input.substr(peg$currPos, 4) === peg$c457) {
                                  s1 = peg$c457;
                                  peg$currPos += 4;
                                } else {
                                  s1 = peg$FAILED;
                                  if (peg$silentFails === 0) { peg$fail(peg$c458); }
                                }
                                if (s1 === peg$FAILED) {
                                  if (input.substr(peg$currPos, 5) === peg$c459) {
                                    s1 = peg$c459;
                                    peg$currPos += 5;
                                  } else {
                                    s1 = peg$FAILED;
                                    if (peg$silentFails === 0) { peg$fail(peg$c460); }
                                  }
                                  if (s1 === peg$FAILED) {
                                    if (input.substr(peg$currPos, 2) === peg$c461) {
                                      s1 = peg$c461;
                                      peg$currPos += 2;
                                    } else {
                                      s1 = peg$FAILED;
                                      if (peg$silentFails === 0) { peg$fail(peg$c462); }
                                    }
                                    if (s1 === peg$FAILED) {
                                      if (input.substr(peg$currPos, 3) === peg$c463) {
                                        s1 = peg$c463;
                                        peg$currPos += 3;
                                      } else {
                                        s1 = peg$FAILED;
                                        if (peg$silentFails === 0) { peg$fail(peg$c464); }
                                      }
                                      if (s1 === peg$FAILED) {
                                        if (input.substr(peg$currPos, 4) === peg$c11) {
                                          s1 = peg$c11;
                                          peg$currPos += 4;
                                        } else {
                                          s1 = peg$FAILED;
                                          if (peg$silentFails === 0) { peg$fail(peg$c12); }
                                        }
                                        if (s1 === peg$FAILED) {
                                          if (input.substr(peg$currPos, 4) === peg$c402) {
                                            s1 = peg$c402;
                                            peg$currPos += 4;
                                          } else {
                                            s1 = peg$FAILED;
                                            if (peg$silentFails === 0) { peg$fail(peg$c403); }
                                          }
                                        }
                                      }
                                    }
                                  }
                                }
                              }
                            }
                          }
                        }
                      }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c465();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseTypeFieldList() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseTypeField();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseTypeFieldListTail();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseTypeFieldListTail();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c281(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$c108;
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c64();
      }
      s0 = s1;
    }

    return s0;
  }

  function peg$parseTypeFieldListTail() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parse__();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 44) {
        s2 = peg$c26;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c27); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parse__();
        if (s3 !== peg$FAILED) {
          s4 = peg$parseTypeField();
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c277(s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseTypeField() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parseFieldName();
    if (s1 !== peg$FAILED) {
      s2 = peg$parse__();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 58) {
          s3 = peg$c20;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c21); }
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$parse__();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseType();
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c466(s1, s5);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFieldName() {
    var s0;

    s0 = peg$parseIdentifierName();
    if (s0 === peg$FAILED) {
      s0 = peg$parseQuotedString();
    }

    return s0;
  }

  function peg$parseAndToken() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 3) === peg$c467) {
      s1 = peg$c467;
      peg$currPos += 3;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c468); }
    }
    if (s1 === peg$FAILED) {
      if (input.substr(peg$currPos, 3) === peg$c469) {
        s1 = peg$c469;
        peg$currPos += 3;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c470); }
      }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseIdentifierRest();
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c471();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseOrToken() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c472) {
      s1 = peg$c472;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c473); }
    }
    if (s1 === peg$FAILED) {
      if (input.substr(peg$currPos, 2) === peg$c474) {
        s1 = peg$c474;
        peg$currPos += 2;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c475); }
      }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseIdentifierRest();
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c476();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseNotToken() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 3) === peg$c300) {
      s1 = peg$c300;
      peg$currPos += 3;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c301); }
    }
    if (s1 === peg$FAILED) {
      if (input.substr(peg$currPos, 3) === peg$c478) {
        s1 = peg$c478;
        peg$currPos += 3;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c479); }
      }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseIdentifierRest();
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c480();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseByToken() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c370) {
      s1 = peg$c370;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c481); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseIdentifierRest();
      peg$silentFails--;
      if (s3 === peg$FAILED) {
        s2 = void 0;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c372();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseIdentifierStart() {
    var s0;

    if (peg$c482.test(input.charAt(peg$currPos))) {
      s0 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c483); }
    }

    return s0;
  }

  function peg$parseIdentifierRest() {
    var s0;

    s0 = peg$parseIdentifierStart();
    if (s0 === peg$FAILED) {
      if (peg$c484.test(input.charAt(peg$currPos))) {
        s0 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s0 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c485); }
      }
    }

    return s0;
  }

  function peg$parseIdentifier() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseIdentifierName();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c486(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseIdentifierName() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$currPos;
    peg$silentFails++;
    s2 = peg$currPos;
    s3 = peg$parseIDGuard();
    if (s3 !== peg$FAILED) {
      s4 = peg$currPos;
      peg$silentFails++;
      s5 = peg$parseIdentifierRest();
      peg$silentFails--;
      if (s5 === peg$FAILED) {
        s4 = void 0;
      } else {
        peg$currPos = s4;
        s4 = peg$FAILED;
      }
      if (s4 !== peg$FAILED) {
        s3 = [s3, s4];
        s2 = s3;
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
    } else {
      peg$currPos = s2;
      s2 = peg$FAILED;
    }
    peg$silentFails--;
    if (s2 === peg$FAILED) {
      s1 = void 0;
    } else {
      peg$currPos = s1;
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseIdentifierStart();
      if (s2 !== peg$FAILED) {
        s3 = [];
        s4 = peg$parseIdentifierRest();
        while (s4 !== peg$FAILED) {
          s3.push(s4);
          s4 = peg$parseIdentifierRest();
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c82();
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 36) {
        s1 = peg$c487;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c488); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c82();
      }
      s0 = s1;
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        if (input.charCodeAt(peg$currPos) === 92) {
          s1 = peg$c424;
          peg$currPos++;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c425); }
        }
        if (s1 !== peg$FAILED) {
          s2 = peg$parseIDGuard();
          if (s2 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c236(s2);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          if (input.substr(peg$currPos, 4) === peg$c11) {
            s1 = peg$c11;
            peg$currPos += 4;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c12); }
          }
          if (s1 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c82();
          }
          s0 = s1;
          if (s0 === peg$FAILED) {
            s0 = peg$currPos;
            s1 = peg$parseSQLTokenSentinels();
            if (s1 !== peg$FAILED) {
              s2 = peg$currPos;
              peg$silentFails++;
              s3 = peg$currPos;
              s4 = peg$parse__();
              if (s4 !== peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 40) {
                  s5 = peg$c16;
                  peg$currPos++;
                } else {
                  s5 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c17); }
                }
                if (s5 !== peg$FAILED) {
                  s4 = [s4, s5];
                  s3 = s4;
                } else {
                  peg$currPos = s3;
                  s3 = peg$FAILED;
                }
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
              peg$silentFails--;
              if (s3 !== peg$FAILED) {
                peg$currPos = s2;
                s2 = void 0;
              } else {
                s2 = peg$FAILED;
              }
              if (s2 !== peg$FAILED) {
                peg$savedPos = s0;
                s1 = peg$c236(s1);
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          }
        }
      }
    }

    return s0;
  }

  function peg$parseIdentifierNames() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseIdentifierName();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parse__();
      if (s4 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 44) {
          s5 = peg$c26;
          peg$currPos++;
        } else {
          s5 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c27); }
        }
        if (s5 !== peg$FAILED) {
          s6 = peg$parse__();
          if (s6 !== peg$FAILED) {
            s7 = peg$parseIdentifierName();
            if (s7 !== peg$FAILED) {
              peg$savedPos = s3;
              s4 = peg$c489(s1, s7);
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$parse__();
        if (s4 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 44) {
            s5 = peg$c26;
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c27); }
          }
          if (s5 !== peg$FAILED) {
            s6 = peg$parse__();
            if (s6 !== peg$FAILED) {
              s7 = peg$parseIdentifierName();
              if (s7 !== peg$FAILED) {
                peg$savedPos = s3;
                s4 = peg$c489(s1, s7);
                s3 = s4;
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c281(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseIDGuard() {
    var s0;

    s0 = peg$parseBooleanLiteral();
    if (s0 === peg$FAILED) {
      s0 = peg$parseNullLiteral();
      if (s0 === peg$FAILED) {
        s0 = peg$parseNaN();
        if (s0 === peg$FAILED) {
          s0 = peg$parseInfinity();
        }
      }
    }

    return s0;
  }

  function peg$parseTime() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseFullDate();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 84) {
        s2 = peg$c490;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c491); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseFullTime();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c492();
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFullDate() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$parseD4();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 45) {
        s2 = peg$c292;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c293); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseD2();
        if (s3 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 45) {
            s4 = peg$c292;
            peg$currPos++;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c293); }
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parseD2();
            if (s5 !== peg$FAILED) {
              s1 = [s1, s2, s3, s4, s5];
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseD4() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    if (peg$c484.test(input.charAt(peg$currPos))) {
      s1 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c485); }
    }
    if (s1 !== peg$FAILED) {
      if (peg$c484.test(input.charAt(peg$currPos))) {
        s2 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c485); }
      }
      if (s2 !== peg$FAILED) {
        if (peg$c484.test(input.charAt(peg$currPos))) {
          s3 = input.charAt(peg$currPos);
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c485); }
        }
        if (s3 !== peg$FAILED) {
          if (peg$c484.test(input.charAt(peg$currPos))) {
            s4 = input.charAt(peg$currPos);
            peg$currPos++;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c485); }
          }
          if (s4 !== peg$FAILED) {
            s1 = [s1, s2, s3, s4];
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseD2() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (peg$c484.test(input.charAt(peg$currPos))) {
      s1 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c485); }
    }
    if (s1 !== peg$FAILED) {
      if (peg$c484.test(input.charAt(peg$currPos))) {
        s2 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c485); }
      }
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFullTime() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parsePartialTime();
    if (s1 !== peg$FAILED) {
      s2 = peg$parseTimeOffset();
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parsePartialTime() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9;

    s0 = peg$currPos;
    s1 = peg$parseD2();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 58) {
        s2 = peg$c20;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c21); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseD2();
        if (s3 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 58) {
            s4 = peg$c20;
            peg$currPos++;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c21); }
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parseD2();
            if (s5 !== peg$FAILED) {
              s6 = peg$currPos;
              if (input.charCodeAt(peg$currPos) === 46) {
                s7 = peg$c116;
                peg$currPos++;
              } else {
                s7 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c117); }
              }
              if (s7 !== peg$FAILED) {
                s8 = [];
                if (peg$c484.test(input.charAt(peg$currPos))) {
                  s9 = input.charAt(peg$currPos);
                  peg$currPos++;
                } else {
                  s9 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c485); }
                }
                if (s9 !== peg$FAILED) {
                  while (s9 !== peg$FAILED) {
                    s8.push(s9);
                    if (peg$c484.test(input.charAt(peg$currPos))) {
                      s9 = input.charAt(peg$currPos);
                      peg$currPos++;
                    } else {
                      s9 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c485); }
                    }
                  }
                } else {
                  s8 = peg$FAILED;
                }
                if (s8 !== peg$FAILED) {
                  s7 = [s7, s8];
                  s6 = s7;
                } else {
                  peg$currPos = s6;
                  s6 = peg$FAILED;
                }
              } else {
                peg$currPos = s6;
                s6 = peg$FAILED;
              }
              if (s6 === peg$FAILED) {
                s6 = null;
              }
              if (s6 !== peg$FAILED) {
                s1 = [s1, s2, s3, s4, s5, s6];
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseTimeOffset() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8;

    if (input.charCodeAt(peg$currPos) === 90) {
      s0 = peg$c493;
      peg$currPos++;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c494); }
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 43) {
        s1 = peg$c290;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c291); }
      }
      if (s1 === peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 45) {
          s1 = peg$c292;
          peg$currPos++;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c293); }
        }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseD2();
        if (s2 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 58) {
            s3 = peg$c20;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c21); }
          }
          if (s3 !== peg$FAILED) {
            s4 = peg$parseD2();
            if (s4 !== peg$FAILED) {
              s5 = peg$currPos;
              if (input.charCodeAt(peg$currPos) === 46) {
                s6 = peg$c116;
                peg$currPos++;
              } else {
                s6 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c117); }
              }
              if (s6 !== peg$FAILED) {
                s7 = [];
                if (peg$c484.test(input.charAt(peg$currPos))) {
                  s8 = input.charAt(peg$currPos);
                  peg$currPos++;
                } else {
                  s8 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c485); }
                }
                if (s8 !== peg$FAILED) {
                  while (s8 !== peg$FAILED) {
                    s7.push(s8);
                    if (peg$c484.test(input.charAt(peg$currPos))) {
                      s8 = input.charAt(peg$currPos);
                      peg$currPos++;
                    } else {
                      s8 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c485); }
                    }
                  }
                } else {
                  s7 = peg$FAILED;
                }
                if (s7 !== peg$FAILED) {
                  s6 = [s6, s7];
                  s5 = s6;
                } else {
                  peg$currPos = s5;
                  s5 = peg$FAILED;
                }
              } else {
                peg$currPos = s5;
                s5 = peg$FAILED;
              }
              if (s5 === peg$FAILED) {
                s5 = null;
              }
              if (s5 !== peg$FAILED) {
                s1 = [s1, s2, s3, s4, s5];
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseDuration() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 45) {
      s1 = peg$c292;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c293); }
    }
    if (s1 === peg$FAILED) {
      s1 = null;
    }
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$parseDecimal();
      if (s4 !== peg$FAILED) {
        s5 = peg$parseTimeUnit();
        if (s5 !== peg$FAILED) {
          s4 = [s4, s5];
          s3 = s4;
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      if (s3 !== peg$FAILED) {
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          s3 = peg$currPos;
          s4 = peg$parseDecimal();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseTimeUnit();
            if (s5 !== peg$FAILED) {
              s4 = [s4, s5];
              s3 = s4;
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        }
      } else {
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c495();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseDecimal() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = peg$parseUInt();
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 46) {
        s3 = peg$c116;
        peg$currPos++;
      } else {
        s3 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c117); }
      }
      if (s3 !== peg$FAILED) {
        s4 = peg$parseUInt();
        if (s4 !== peg$FAILED) {
          s3 = [s3, s4];
          s2 = s3;
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 === peg$FAILED) {
        s2 = null;
      }
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseTimeUnit() {
    var s0;

    if (input.substr(peg$currPos, 2) === peg$c496) {
      s0 = peg$c496;
      peg$currPos += 2;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c497); }
    }
    if (s0 === peg$FAILED) {
      if (input.substr(peg$currPos, 2) === peg$c498) {
        s0 = peg$c498;
        peg$currPos += 2;
      } else {
        s0 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c499); }
      }
      if (s0 === peg$FAILED) {
        if (input.substr(peg$currPos, 2) === peg$c500) {
          s0 = peg$c500;
          peg$currPos += 2;
        } else {
          s0 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c501); }
        }
        if (s0 === peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 115) {
            s0 = peg$c502;
            peg$currPos++;
          } else {
            s0 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c503); }
          }
          if (s0 === peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 109) {
              s0 = peg$c504;
              peg$currPos++;
            } else {
              s0 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c505); }
            }
            if (s0 === peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 104) {
                s0 = peg$c506;
                peg$currPos++;
              } else {
                s0 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c507); }
              }
              if (s0 === peg$FAILED) {
                if (input.charCodeAt(peg$currPos) === 100) {
                  s0 = peg$c508;
                  peg$currPos++;
                } else {
                  s0 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c509); }
                }
                if (s0 === peg$FAILED) {
                  if (input.charCodeAt(peg$currPos) === 119) {
                    s0 = peg$c510;
                    peg$currPos++;
                  } else {
                    s0 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c511); }
                  }
                  if (s0 === peg$FAILED) {
                    if (input.charCodeAt(peg$currPos) === 121) {
                      s0 = peg$c512;
                      peg$currPos++;
                    } else {
                      s0 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c513); }
                    }
                  }
                }
              }
            }
          }
        }
      }
    }

    return s0;
  }

  function peg$parseIP() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$parseUInt();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 46) {
        s2 = peg$c116;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c117); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseUInt();
        if (s3 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 46) {
            s4 = peg$c116;
            peg$currPos++;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c117); }
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parseUInt();
            if (s5 !== peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 46) {
                s6 = peg$c116;
                peg$currPos++;
              } else {
                s6 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c117); }
              }
              if (s6 !== peg$FAILED) {
                s7 = peg$parseUInt();
                if (s7 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c82();
                  s0 = s1;
                } else {
                  peg$currPos = s0;
                  s0 = peg$FAILED;
                }
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseIP6() {
    var s0, s1, s2, s3, s4, s5, s6, s7;

    s0 = peg$currPos;
    s1 = peg$currPos;
    peg$silentFails++;
    s2 = peg$currPos;
    s3 = peg$parseHex();
    if (s3 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 58) {
        s4 = peg$c20;
        peg$currPos++;
      } else {
        s4 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c21); }
      }
      if (s4 !== peg$FAILED) {
        s5 = peg$parseHex();
        if (s5 !== peg$FAILED) {
          s6 = peg$currPos;
          peg$silentFails++;
          s7 = peg$parseHexDigit();
          if (s7 === peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 58) {
              s7 = peg$c20;
              peg$currPos++;
            } else {
              s7 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c21); }
            }
          }
          peg$silentFails--;
          if (s7 === peg$FAILED) {
            s6 = void 0;
          } else {
            peg$currPos = s6;
            s6 = peg$FAILED;
          }
          if (s6 !== peg$FAILED) {
            s3 = [s3, s4, s5, s6];
            s2 = s3;
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
    } else {
      peg$currPos = s2;
      s2 = peg$FAILED;
    }
    peg$silentFails--;
    if (s2 === peg$FAILED) {
      s1 = void 0;
    } else {
      peg$currPos = s1;
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseIP6Variations();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c5(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseIP6Variations() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = [];
    s2 = peg$parseHexColon();
    if (s2 !== peg$FAILED) {
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parseHexColon();
      }
    } else {
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseIP6Tail();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c514(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = peg$parseHex();
      if (s1 !== peg$FAILED) {
        s2 = [];
        s3 = peg$parseColonHex();
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          s3 = peg$parseColonHex();
        }
        if (s2 !== peg$FAILED) {
          if (input.substr(peg$currPos, 2) === peg$c515) {
            s3 = peg$c515;
            peg$currPos += 2;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c516); }
          }
          if (s3 !== peg$FAILED) {
            s4 = [];
            s5 = peg$parseHexColon();
            while (s5 !== peg$FAILED) {
              s4.push(s5);
              s5 = peg$parseHexColon();
            }
            if (s4 !== peg$FAILED) {
              s5 = peg$parseIP6Tail();
              if (s5 !== peg$FAILED) {
                peg$savedPos = s0;
                s1 = peg$c517(s1, s2, s4, s5);
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        if (input.substr(peg$currPos, 2) === peg$c515) {
          s1 = peg$c515;
          peg$currPos += 2;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c516); }
        }
        if (s1 !== peg$FAILED) {
          s2 = [];
          s3 = peg$parseHexColon();
          while (s3 !== peg$FAILED) {
            s2.push(s3);
            s3 = peg$parseHexColon();
          }
          if (s2 !== peg$FAILED) {
            s3 = peg$parseIP6Tail();
            if (s3 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c518(s2, s3);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          s1 = peg$parseHex();
          if (s1 !== peg$FAILED) {
            s2 = [];
            s3 = peg$parseColonHex();
            while (s3 !== peg$FAILED) {
              s2.push(s3);
              s3 = peg$parseColonHex();
            }
            if (s2 !== peg$FAILED) {
              if (input.substr(peg$currPos, 2) === peg$c515) {
                s3 = peg$c515;
                peg$currPos += 2;
              } else {
                s3 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c516); }
              }
              if (s3 !== peg$FAILED) {
                peg$savedPos = s0;
                s1 = peg$c519(s1, s2);
                s0 = s1;
              } else {
                peg$currPos = s0;
                s0 = peg$FAILED;
              }
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
          if (s0 === peg$FAILED) {
            s0 = peg$currPos;
            if (input.substr(peg$currPos, 2) === peg$c515) {
              s1 = peg$c515;
              peg$currPos += 2;
            } else {
              s1 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c516); }
            }
            if (s1 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c520();
            }
            s0 = s1;
          }
        }
      }
    }

    return s0;
  }

  function peg$parseIP6Tail() {
    var s0;

    s0 = peg$parseIP();
    if (s0 === peg$FAILED) {
      s0 = peg$parseHex();
    }

    return s0;
  }

  function peg$parseColonHex() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 58) {
      s1 = peg$c20;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c21); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseHex();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c521(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseHexColon() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$parseHex();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 58) {
        s2 = peg$c20;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c21); }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c522(s1);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseIP4Net() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseIP();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 47) {
        s2 = peg$c294;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c295); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseUInt();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c523(s1, s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseIP6Net() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseIP6();
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 47) {
        s2 = peg$c294;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c295); }
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseUInt();
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c524(s1, s3);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseUInt() {
    var s0, s1;

    s0 = peg$currPos;
    s1 = peg$parseUIntString();
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c525(s1);
    }
    s0 = s1;

    return s0;
  }

  function peg$parseIntString() {
    var s0;

    s0 = peg$parseUIntString();
    if (s0 === peg$FAILED) {
      s0 = peg$parseMinusIntString();
    }

    return s0;
  }

  function peg$parseUIntString() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = [];
    if (peg$c484.test(input.charAt(peg$currPos))) {
      s2 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s2 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c485); }
    }
    if (s2 !== peg$FAILED) {
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        if (peg$c484.test(input.charAt(peg$currPos))) {
          s2 = input.charAt(peg$currPos);
          peg$currPos++;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c485); }
        }
      }
    } else {
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c82();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseMinusIntString() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 45) {
      s1 = peg$c292;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c293); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseUIntString();
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c82();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseFloatString() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 45) {
      s1 = peg$c292;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c293); }
    }
    if (s1 === peg$FAILED) {
      s1 = null;
    }
    if (s1 !== peg$FAILED) {
      s2 = [];
      if (peg$c484.test(input.charAt(peg$currPos))) {
        s3 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s3 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c485); }
      }
      if (s3 !== peg$FAILED) {
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          if (peg$c484.test(input.charAt(peg$currPos))) {
            s3 = input.charAt(peg$currPos);
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c485); }
          }
        }
      } else {
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 46) {
          s3 = peg$c116;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c117); }
        }
        if (s3 !== peg$FAILED) {
          s4 = [];
          if (peg$c484.test(input.charAt(peg$currPos))) {
            s5 = input.charAt(peg$currPos);
            peg$currPos++;
          } else {
            s5 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c485); }
          }
          while (s5 !== peg$FAILED) {
            s4.push(s5);
            if (peg$c484.test(input.charAt(peg$currPos))) {
              s5 = input.charAt(peg$currPos);
              peg$currPos++;
            } else {
              s5 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c485); }
            }
          }
          if (s4 !== peg$FAILED) {
            s5 = peg$parseExponentPart();
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c526();
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 45) {
        s1 = peg$c292;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c293); }
      }
      if (s1 === peg$FAILED) {
        s1 = null;
      }
      if (s1 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 46) {
          s2 = peg$c116;
          peg$currPos++;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c117); }
        }
        if (s2 !== peg$FAILED) {
          s3 = [];
          if (peg$c484.test(input.charAt(peg$currPos))) {
            s4 = input.charAt(peg$currPos);
            peg$currPos++;
          } else {
            s4 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c485); }
          }
          if (s4 !== peg$FAILED) {
            while (s4 !== peg$FAILED) {
              s3.push(s4);
              if (peg$c484.test(input.charAt(peg$currPos))) {
                s4 = input.charAt(peg$currPos);
                peg$currPos++;
              } else {
                s4 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c485); }
              }
            }
          } else {
            s3 = peg$FAILED;
          }
          if (s3 !== peg$FAILED) {
            s4 = peg$parseExponentPart();
            if (s4 === peg$FAILED) {
              s4 = null;
            }
            if (s4 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c526();
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        s1 = peg$parseNaN();
        if (s1 === peg$FAILED) {
          s1 = peg$parseInfinity();
        }
        if (s1 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c82();
        }
        s0 = s1;
      }
    }

    return s0;
  }

  function peg$parseExponentPart() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 1).toLowerCase() === peg$c527) {
      s1 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c528); }
    }
    if (s1 !== peg$FAILED) {
      if (peg$c529.test(input.charAt(peg$currPos))) {
        s2 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c530); }
      }
      if (s2 === peg$FAILED) {
        s2 = null;
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseUIntString();
        if (s3 !== peg$FAILED) {
          s1 = [s1, s2, s3];
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseNaN() {
    var s0;

    if (input.substr(peg$currPos, 3) === peg$c531) {
      s0 = peg$c531;
      peg$currPos += 3;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c532); }
    }

    return s0;
  }

  function peg$parseInfinity() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 45) {
      s1 = peg$c292;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c293); }
    }
    if (s1 === peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 43) {
        s1 = peg$c290;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c291); }
      }
    }
    if (s1 === peg$FAILED) {
      s1 = null;
    }
    if (s1 !== peg$FAILED) {
      if (input.substr(peg$currPos, 3) === peg$c533) {
        s2 = peg$c533;
        peg$currPos += 3;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c534); }
      }
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseHex() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = [];
    s2 = peg$parseHexDigit();
    if (s2 !== peg$FAILED) {
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        s2 = peg$parseHexDigit();
      }
    } else {
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c82();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseHexDigit() {
    var s0;

    if (peg$c535.test(input.charAt(peg$currPos))) {
      s0 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c536); }
    }

    return s0;
  }

  function peg$parseQuotedString() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 34) {
      s1 = peg$c419;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c420); }
    }
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseDoubleQuotedChar();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseDoubleQuotedChar();
      }
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 34) {
          s3 = peg$c419;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c420); }
        }
        if (s3 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c537(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 39) {
        s1 = peg$c421;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c422); }
      }
      if (s1 !== peg$FAILED) {
        s2 = [];
        s3 = peg$parseSingleQuotedChar();
        while (s3 !== peg$FAILED) {
          s2.push(s3);
          s3 = peg$parseSingleQuotedChar();
        }
        if (s2 !== peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 39) {
            s3 = peg$c421;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c422); }
          }
          if (s3 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c537(s2);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseDoubleQuotedChar() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$currPos;
    peg$silentFails++;
    if (input.charCodeAt(peg$currPos) === 34) {
      s2 = peg$c419;
      peg$currPos++;
    } else {
      s2 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c420); }
    }
    if (s2 === peg$FAILED) {
      s2 = peg$parseEscapedChar();
    }
    peg$silentFails--;
    if (s2 === peg$FAILED) {
      s1 = void 0;
    } else {
      peg$currPos = s1;
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      if (input.length > peg$currPos) {
        s2 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c538); }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c82();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 92) {
        s1 = peg$c424;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c425); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseEscapeSequence();
        if (s2 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c197(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseKeyWord() {
    var s0, s1, s2, s3;

    s0 = peg$currPos;
    s1 = peg$parseKeyWordStart();
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$parseKeyWordRest();
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$parseKeyWordRest();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c539(s1, s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseKeyWordStart() {
    var s0;

    s0 = peg$parseKeyWordChars();
    if (s0 === peg$FAILED) {
      s0 = peg$parseKeyWordEsc();
    }

    return s0;
  }

  function peg$parseKeyWordChars() {
    var s0, s1;

    s0 = peg$currPos;
    if (peg$c540.test(input.charAt(peg$currPos))) {
      s1 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c541); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c82();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseKeyWordRest() {
    var s0;

    s0 = peg$parseKeyWordStart();
    if (s0 === peg$FAILED) {
      if (peg$c484.test(input.charAt(peg$currPos))) {
        s0 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s0 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c485); }
      }
    }

    return s0;
  }

  function peg$parseKeyWordEsc() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 92) {
      s1 = peg$c424;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c425); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseKeywordEscape();
      if (s2 === peg$FAILED) {
        s2 = peg$parseEscapeSequence();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c197(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseGlobPattern() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    s1 = peg$currPos;
    peg$silentFails++;
    s2 = peg$parseGlobProperStart();
    peg$silentFails--;
    if (s2 !== peg$FAILED) {
      peg$currPos = s1;
      s1 = void 0;
    } else {
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      peg$silentFails++;
      s3 = peg$parseGlobHasStar();
      peg$silentFails--;
      if (s3 !== peg$FAILED) {
        peg$currPos = s2;
        s2 = void 0;
      } else {
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        s3 = peg$parseGlobStart();
        if (s3 !== peg$FAILED) {
          s4 = [];
          s5 = peg$parseGlobRest();
          while (s5 !== peg$FAILED) {
            s4.push(s5);
            s5 = peg$parseGlobRest();
          }
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c542(s3, s4);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseGlobProperStart() {
    var s0, s1, s2;

    s0 = peg$parseKeyWordStart();
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      s1 = [];
      if (input.charCodeAt(peg$currPos) === 42) {
        s2 = peg$c90;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c91); }
      }
      if (s2 !== peg$FAILED) {
        while (s2 !== peg$FAILED) {
          s1.push(s2);
          if (input.charCodeAt(peg$currPos) === 42) {
            s2 = peg$c90;
            peg$currPos++;
          } else {
            s2 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c91); }
          }
        }
      } else {
        s1 = peg$FAILED;
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseKeyWordRest();
        if (s2 !== peg$FAILED) {
          s1 = [s1, s2];
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseGlobHasStar() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = [];
    s2 = peg$parseKeyWordRest();
    while (s2 !== peg$FAILED) {
      s1.push(s2);
      s2 = peg$parseKeyWordRest();
    }
    if (s1 !== peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 42) {
        s2 = peg$c90;
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c91); }
      }
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseGlobStart() {
    var s0, s1;

    s0 = peg$parseKeyWordChars();
    if (s0 === peg$FAILED) {
      s0 = peg$parseGlobEsc();
      if (s0 === peg$FAILED) {
        s0 = peg$currPos;
        if (input.charCodeAt(peg$currPos) === 42) {
          s1 = peg$c90;
          peg$currPos++;
        } else {
          s1 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c91); }
        }
        if (s1 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c543();
        }
        s0 = s1;
      }
    }

    return s0;
  }

  function peg$parseGlobRest() {
    var s0;

    s0 = peg$parseGlobStart();
    if (s0 === peg$FAILED) {
      if (peg$c484.test(input.charAt(peg$currPos))) {
        s0 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s0 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c485); }
      }
    }

    return s0;
  }

  function peg$parseGlobEsc() {
    var s0, s1, s2;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 92) {
      s1 = peg$c424;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c425); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseGlobEscape();
      if (s2 === peg$FAILED) {
        s2 = peg$parseEscapeSequence();
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c197(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseGlobEscape() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 61) {
      s1 = peg$c8;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c9); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c544();
    }
    s0 = s1;
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 42) {
        s1 = peg$c90;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c91); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c545();
      }
      s0 = s1;
      if (s0 === peg$FAILED) {
        if (peg$c529.test(input.charAt(peg$currPos))) {
          s0 = input.charAt(peg$currPos);
          peg$currPos++;
        } else {
          s0 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c530); }
        }
      }
    }

    return s0;
  }

  function peg$parseSingleQuotedChar() {
    var s0, s1, s2;

    s0 = peg$currPos;
    s1 = peg$currPos;
    peg$silentFails++;
    if (input.charCodeAt(peg$currPos) === 39) {
      s2 = peg$c421;
      peg$currPos++;
    } else {
      s2 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c422); }
    }
    if (s2 === peg$FAILED) {
      s2 = peg$parseEscapedChar();
    }
    peg$silentFails--;
    if (s2 === peg$FAILED) {
      s1 = void 0;
    } else {
      peg$currPos = s1;
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      if (input.length > peg$currPos) {
        s2 = input.charAt(peg$currPos);
        peg$currPos++;
      } else {
        s2 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c538); }
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c82();
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 92) {
        s1 = peg$c424;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c425); }
      }
      if (s1 !== peg$FAILED) {
        s2 = peg$parseEscapeSequence();
        if (s2 !== peg$FAILED) {
          peg$savedPos = s0;
          s1 = peg$c197(s2);
          s0 = s1;
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseEscapeSequence() {
    var s0;

    s0 = peg$parseSingleCharEscape();
    if (s0 === peg$FAILED) {
      s0 = peg$parseUnicodeEscape();
    }

    return s0;
  }

  function peg$parseSingleCharEscape() {
    var s0, s1;

    if (input.charCodeAt(peg$currPos) === 39) {
      s0 = peg$c421;
      peg$currPos++;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c422); }
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 34) {
        s1 = peg$c419;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c420); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c82();
      }
      s0 = s1;
      if (s0 === peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 92) {
          s0 = peg$c424;
          peg$currPos++;
        } else {
          s0 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c425); }
        }
        if (s0 === peg$FAILED) {
          s0 = peg$currPos;
          if (input.charCodeAt(peg$currPos) === 98) {
            s1 = peg$c546;
            peg$currPos++;
          } else {
            s1 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c547); }
          }
          if (s1 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c548();
          }
          s0 = s1;
          if (s0 === peg$FAILED) {
            s0 = peg$currPos;
            if (input.charCodeAt(peg$currPos) === 102) {
              s1 = peg$c549;
              peg$currPos++;
            } else {
              s1 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c550); }
            }
            if (s1 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c551();
            }
            s0 = s1;
            if (s0 === peg$FAILED) {
              s0 = peg$currPos;
              if (input.charCodeAt(peg$currPos) === 110) {
                s1 = peg$c552;
                peg$currPos++;
              } else {
                s1 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c553); }
              }
              if (s1 !== peg$FAILED) {
                peg$savedPos = s0;
                s1 = peg$c554();
              }
              s0 = s1;
              if (s0 === peg$FAILED) {
                s0 = peg$currPos;
                if (input.charCodeAt(peg$currPos) === 114) {
                  s1 = peg$c555;
                  peg$currPos++;
                } else {
                  s1 = peg$FAILED;
                  if (peg$silentFails === 0) { peg$fail(peg$c556); }
                }
                if (s1 !== peg$FAILED) {
                  peg$savedPos = s0;
                  s1 = peg$c557();
                }
                s0 = s1;
                if (s0 === peg$FAILED) {
                  s0 = peg$currPos;
                  if (input.charCodeAt(peg$currPos) === 116) {
                    s1 = peg$c558;
                    peg$currPos++;
                  } else {
                    s1 = peg$FAILED;
                    if (peg$silentFails === 0) { peg$fail(peg$c559); }
                  }
                  if (s1 !== peg$FAILED) {
                    peg$savedPos = s0;
                    s1 = peg$c560();
                  }
                  s0 = s1;
                  if (s0 === peg$FAILED) {
                    s0 = peg$currPos;
                    if (input.charCodeAt(peg$currPos) === 118) {
                      s1 = peg$c561;
                      peg$currPos++;
                    } else {
                      s1 = peg$FAILED;
                      if (peg$silentFails === 0) { peg$fail(peg$c562); }
                    }
                    if (s1 !== peg$FAILED) {
                      peg$savedPos = s0;
                      s1 = peg$c563();
                    }
                    s0 = s1;
                  }
                }
              }
            }
          }
        }
      }
    }

    return s0;
  }

  function peg$parseKeywordEscape() {
    var s0, s1;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 61) {
      s1 = peg$c8;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c9); }
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c544();
    }
    s0 = s1;
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 42) {
        s1 = peg$c90;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c91); }
      }
      if (s1 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c543();
      }
      s0 = s1;
      if (s0 === peg$FAILED) {
        if (peg$c529.test(input.charAt(peg$currPos))) {
          s0 = input.charAt(peg$currPos);
          peg$currPos++;
        } else {
          s0 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c530); }
        }
      }
    }

    return s0;
  }

  function peg$parseUnicodeEscape() {
    var s0, s1, s2, s3, s4, s5, s6, s7, s8, s9;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 117) {
      s1 = peg$c564;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c565); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$currPos;
      s3 = peg$parseHexDigit();
      if (s3 !== peg$FAILED) {
        s4 = peg$parseHexDigit();
        if (s4 !== peg$FAILED) {
          s5 = peg$parseHexDigit();
          if (s5 !== peg$FAILED) {
            s6 = peg$parseHexDigit();
            if (s6 !== peg$FAILED) {
              s3 = [s3, s4, s5, s6];
              s2 = s3;
            } else {
              peg$currPos = s2;
              s2 = peg$FAILED;
            }
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
      if (s2 !== peg$FAILED) {
        peg$savedPos = s0;
        s1 = peg$c566(s2);
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }
    if (s0 === peg$FAILED) {
      s0 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 117) {
        s1 = peg$c564;
        peg$currPos++;
      } else {
        s1 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c565); }
      }
      if (s1 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 123) {
          s2 = peg$c54;
          peg$currPos++;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c55); }
        }
        if (s2 !== peg$FAILED) {
          s3 = peg$currPos;
          s4 = peg$parseHexDigit();
          if (s4 !== peg$FAILED) {
            s5 = peg$parseHexDigit();
            if (s5 === peg$FAILED) {
              s5 = null;
            }
            if (s5 !== peg$FAILED) {
              s6 = peg$parseHexDigit();
              if (s6 === peg$FAILED) {
                s6 = null;
              }
              if (s6 !== peg$FAILED) {
                s7 = peg$parseHexDigit();
                if (s7 === peg$FAILED) {
                  s7 = null;
                }
                if (s7 !== peg$FAILED) {
                  s8 = peg$parseHexDigit();
                  if (s8 === peg$FAILED) {
                    s8 = null;
                  }
                  if (s8 !== peg$FAILED) {
                    s9 = peg$parseHexDigit();
                    if (s9 === peg$FAILED) {
                      s9 = null;
                    }
                    if (s9 !== peg$FAILED) {
                      s4 = [s4, s5, s6, s7, s8, s9];
                      s3 = s4;
                    } else {
                      peg$currPos = s3;
                      s3 = peg$FAILED;
                    }
                  } else {
                    peg$currPos = s3;
                    s3 = peg$FAILED;
                  }
                } else {
                  peg$currPos = s3;
                  s3 = peg$FAILED;
                }
              } else {
                peg$currPos = s3;
                s3 = peg$FAILED;
              }
            } else {
              peg$currPos = s3;
              s3 = peg$FAILED;
            }
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
          if (s3 !== peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 125) {
              s4 = peg$c326;
              peg$currPos++;
            } else {
              s4 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c327); }
            }
            if (s4 !== peg$FAILED) {
              peg$savedPos = s0;
              s1 = peg$c566(s3);
              s0 = s1;
            } else {
              peg$currPos = s0;
              s0 = peg$FAILED;
            }
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    }

    return s0;
  }

  function peg$parseRegexpPattern() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.charCodeAt(peg$currPos) === 47) {
      s1 = peg$c294;
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c295); }
    }
    if (s1 !== peg$FAILED) {
      s2 = peg$parseRegexpBody();
      if (s2 !== peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 47) {
          s3 = peg$c294;
          peg$currPos++;
        } else {
          s3 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c295); }
        }
        if (s3 !== peg$FAILED) {
          s4 = peg$currPos;
          peg$silentFails++;
          s5 = peg$parseKeyWordStart();
          peg$silentFails--;
          if (s5 === peg$FAILED) {
            s4 = void 0;
          } else {
            peg$currPos = s4;
            s4 = peg$FAILED;
          }
          if (s4 !== peg$FAILED) {
            peg$savedPos = s0;
            s1 = peg$c221(s2);
            s0 = s1;
          } else {
            peg$currPos = s0;
            s0 = peg$FAILED;
          }
        } else {
          peg$currPos = s0;
          s0 = peg$FAILED;
        }
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseRegexpBody() {
    var s0, s1, s2, s3, s4;

    s0 = peg$currPos;
    s1 = [];
    if (peg$c567.test(input.charAt(peg$currPos))) {
      s2 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s2 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c568); }
    }
    if (s2 === peg$FAILED) {
      s2 = peg$currPos;
      if (input.charCodeAt(peg$currPos) === 92) {
        s3 = peg$c424;
        peg$currPos++;
      } else {
        s3 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c425); }
      }
      if (s3 !== peg$FAILED) {
        if (input.length > peg$currPos) {
          s4 = input.charAt(peg$currPos);
          peg$currPos++;
        } else {
          s4 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c538); }
        }
        if (s4 !== peg$FAILED) {
          s3 = [s3, s4];
          s2 = s3;
        } else {
          peg$currPos = s2;
          s2 = peg$FAILED;
        }
      } else {
        peg$currPos = s2;
        s2 = peg$FAILED;
      }
    }
    if (s2 !== peg$FAILED) {
      while (s2 !== peg$FAILED) {
        s1.push(s2);
        if (peg$c567.test(input.charAt(peg$currPos))) {
          s2 = input.charAt(peg$currPos);
          peg$currPos++;
        } else {
          s2 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c568); }
        }
        if (s2 === peg$FAILED) {
          s2 = peg$currPos;
          if (input.charCodeAt(peg$currPos) === 92) {
            s3 = peg$c424;
            peg$currPos++;
          } else {
            s3 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c425); }
          }
          if (s3 !== peg$FAILED) {
            if (input.length > peg$currPos) {
              s4 = input.charAt(peg$currPos);
              peg$currPos++;
            } else {
              s4 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c538); }
            }
            if (s4 !== peg$FAILED) {
              s3 = [s3, s4];
              s2 = s3;
            } else {
              peg$currPos = s2;
              s2 = peg$FAILED;
            }
          } else {
            peg$currPos = s2;
            s2 = peg$FAILED;
          }
        }
      }
    } else {
      s1 = peg$FAILED;
    }
    if (s1 !== peg$FAILED) {
      peg$savedPos = s0;
      s1 = peg$c82();
    }
    s0 = s1;

    return s0;
  }

  function peg$parseEscapedChar() {
    var s0;

    if (peg$c569.test(input.charAt(peg$currPos))) {
      s0 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c570); }
    }

    return s0;
  }

  function peg$parse_() {
    var s0, s1;

    s0 = [];
    s1 = peg$parseAnySpace();
    if (s1 !== peg$FAILED) {
      while (s1 !== peg$FAILED) {
        s0.push(s1);
        s1 = peg$parseAnySpace();
      }
    } else {
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parse__() {
    var s0, s1;

    s0 = [];
    s1 = peg$parseAnySpace();
    while (s1 !== peg$FAILED) {
      s0.push(s1);
      s1 = peg$parseAnySpace();
    }

    return s0;
  }

  function peg$parseAnySpace() {
    var s0;

    s0 = peg$parseWhiteSpace();
    if (s0 === peg$FAILED) {
      s0 = peg$parseLineTerminator();
      if (s0 === peg$FAILED) {
        s0 = peg$parseComment();
      }
    }

    return s0;
  }

  function peg$parseSourceCharacter() {
    var s0;

    if (input.length > peg$currPos) {
      s0 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c538); }
    }

    return s0;
  }

  function peg$parseWhiteSpace() {
    var s0;

    peg$silentFails++;
    if (input.charCodeAt(peg$currPos) === 9) {
      s0 = peg$c572;
      peg$currPos++;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c573); }
    }
    if (s0 === peg$FAILED) {
      if (input.charCodeAt(peg$currPos) === 11) {
        s0 = peg$c574;
        peg$currPos++;
      } else {
        s0 = peg$FAILED;
        if (peg$silentFails === 0) { peg$fail(peg$c575); }
      }
      if (s0 === peg$FAILED) {
        if (input.charCodeAt(peg$currPos) === 12) {
          s0 = peg$c576;
          peg$currPos++;
        } else {
          s0 = peg$FAILED;
          if (peg$silentFails === 0) { peg$fail(peg$c577); }
        }
        if (s0 === peg$FAILED) {
          if (input.charCodeAt(peg$currPos) === 32) {
            s0 = peg$c578;
            peg$currPos++;
          } else {
            s0 = peg$FAILED;
            if (peg$silentFails === 0) { peg$fail(peg$c579); }
          }
          if (s0 === peg$FAILED) {
            if (input.charCodeAt(peg$currPos) === 160) {
              s0 = peg$c580;
              peg$currPos++;
            } else {
              s0 = peg$FAILED;
              if (peg$silentFails === 0) { peg$fail(peg$c581); }
            }
            if (s0 === peg$FAILED) {
              if (input.charCodeAt(peg$currPos) === 65279) {
                s0 = peg$c582;
                peg$currPos++;
              } else {
                s0 = peg$FAILED;
                if (peg$silentFails === 0) { peg$fail(peg$c583); }
              }
            }
          }
        }
      }
    }
    peg$silentFails--;
    if (s0 === peg$FAILED) {
      if (peg$silentFails === 0) { peg$fail(peg$c571); }
    }

    return s0;
  }

  function peg$parseLineTerminator() {
    var s0;

    if (peg$c584.test(input.charAt(peg$currPos))) {
      s0 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s0 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c585); }
    }

    return s0;
  }

  function peg$parseComment() {
    var s0;

    peg$silentFails++;
    s0 = peg$parseSingleLineComment();
    peg$silentFails--;
    if (s0 === peg$FAILED) {
      if (peg$silentFails === 0) { peg$fail(peg$c586); }
    }

    return s0;
  }

  function peg$parseSingleLineComment() {
    var s0, s1, s2, s3, s4, s5;

    s0 = peg$currPos;
    if (input.substr(peg$currPos, 2) === peg$c591) {
      s1 = peg$c591;
      peg$currPos += 2;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c592); }
    }
    if (s1 !== peg$FAILED) {
      s2 = [];
      s3 = peg$currPos;
      s4 = peg$currPos;
      peg$silentFails++;
      s5 = peg$parseLineTerminator();
      peg$silentFails--;
      if (s5 === peg$FAILED) {
        s4 = void 0;
      } else {
        peg$currPos = s4;
        s4 = peg$FAILED;
      }
      if (s4 !== peg$FAILED) {
        s5 = peg$parseSourceCharacter();
        if (s5 !== peg$FAILED) {
          s4 = [s4, s5];
          s3 = s4;
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      } else {
        peg$currPos = s3;
        s3 = peg$FAILED;
      }
      while (s3 !== peg$FAILED) {
        s2.push(s3);
        s3 = peg$currPos;
        s4 = peg$currPos;
        peg$silentFails++;
        s5 = peg$parseLineTerminator();
        peg$silentFails--;
        if (s5 === peg$FAILED) {
          s4 = void 0;
        } else {
          peg$currPos = s4;
          s4 = peg$FAILED;
        }
        if (s4 !== peg$FAILED) {
          s5 = peg$parseSourceCharacter();
          if (s5 !== peg$FAILED) {
            s4 = [s4, s5];
            s3 = s4;
          } else {
            peg$currPos = s3;
            s3 = peg$FAILED;
          }
        } else {
          peg$currPos = s3;
          s3 = peg$FAILED;
        }
      }
      if (s2 !== peg$FAILED) {
        s1 = [s1, s2];
        s0 = s1;
      } else {
        peg$currPos = s0;
        s0 = peg$FAILED;
      }
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseEOF() {
    var s0, s1;

    s0 = peg$currPos;
    peg$silentFails++;
    if (input.length > peg$currPos) {
      s1 = input.charAt(peg$currPos);
      peg$currPos++;
    } else {
      s1 = peg$FAILED;
      if (peg$silentFails === 0) { peg$fail(peg$c538); }
    }
    peg$silentFails--;
    if (s1 === peg$FAILED) {
      s0 = void 0;
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function peg$parseEOKW() {
    var s0, s1;

    s0 = peg$currPos;
    peg$silentFails++;
    s1 = peg$parseKeyWordChars();
    peg$silentFails--;
    if (s1 === peg$FAILED) {
      s0 = void 0;
    } else {
      peg$currPos = s0;
      s0 = peg$FAILED;
    }

    return s0;
  }

  function makeArgMap(args) {
    let m = {};
    for (let arg of args) {
      if (arg.name in m) {
        throw new Error(`Duplicate argument -${arg.name}`);
      }
      m[arg.name] = arg.value;
    }
    return m
  }

  function makeBinaryExprChain(first, rest) {
    let ret = first;
    for (let part of rest) {
      ret = { kind: "BinaryExpr", op: part[0], lhs: ret, rhs: part[1] };
    }
    return ret
  }

  function makeTemplateExprChain(args) {
    let ret = args[0];
    for (let part of args.slice(1)) {
      ret = { kind: "BinaryExpr", op: "+", lhs: ret, rhs: part };
    }
    return ret
  }

  function joinChars(chars) {
    return chars.join("");
  }

  function makeUnicodeChar(chars) {
    let n = parseInt(chars.join(""), 16);
    if (n < 0x10000) {
      return String.fromCharCode(n);
    }

    // stupid JavaScript 16 bit code points...
    n -= 0x10000;
    let surrogate1 = 0xD800 + ((n >> 10) & 0x7ff);
    let surrogate2 = 0xDC00 + (n & 0x3ff);
    return String.fromCharCode(surrogate1) + String.fromCharCode(surrogate2);
  }



  peg$result = peg$startRuleFunction();

  if (peg$result !== peg$FAILED && peg$currPos === input.length) {
    return peg$result;
  } else {
    if (peg$result !== peg$FAILED && peg$currPos < input.length) {
      peg$fail(peg$endExpectation());
    }

    throw peg$buildStructuredError(
      peg$maxFailExpected,
      peg$maxFailPos < input.length ? input.charAt(peg$maxFailPos) : null,
      peg$maxFailPos < input.length
        ? peg$computeLocation(peg$maxFailPos, peg$maxFailPos + 1)
        : peg$computeLocation(peg$maxFailPos, peg$maxFailPos)
    );
  }
}

var parser = {
  SyntaxError: peg$SyntaxError,
  parse:       peg$parse
};

export default parser;

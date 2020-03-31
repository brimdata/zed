let reglob = require("../reglob/reglob")

  function makeSequentialProc(procs) {
    return { op: "SequentialProc", procs };
  }

function makeParallelProc(procs) {
  return { op: "ParallelProc", procs };
}

function makeLiteral(type, value) { return { op: "Literal", type, value }; }
function getValueType(v) { return v.type; }

function makeFieldCall(fn, field, param) {
  return { op: "FieldCall", fn, field, param };
}
function chainFieldCalls(base, derefs) {
  let ret = { op: "FieldRead", field: base };
  for (let d of derefs) {
    d.field = ret
    ret = d
  }
  return ret
}

function makeMatchAll() {
  return { op: "MatchAll" };
}

function makeCompareField(comparator, field, value) {
  return { op: "CompareField", comparator, field, value };
}

function makeCompareAny(comparator, recursive, value) {
  return { op: "CompareAny", comparator, recursive, value };
}

function makeLogicalNot(expr) { return { op: "LogicalNot", expr }; }

function makeChain(first, rest, op) {
  if (!rest || rest.length == 0) {
    return first;
  }
  let result = first;
  for (let term of rest) {
    result = { op, left: result, right: term };
  }
  return result;
}

function makeOrChain(first, rest) {
  return makeChain(first, rest, "LogicalOr");
}
function makeAndChain(first, rest) {
  return makeChain(first, rest, "LogicalAnd");
}

function makeArg(name, value) {
  return {name, value};
}

function makeSortProc(args, fields) {
  let argsMap = new Map();
  for (let arg of args) {
    if (argsMap.has(arg.name)) {
      throw new Error(`Duplicate argument -${arg.name}`);
    }
    argsMap.set(arg.name, arg.value);
  }

  let sortdir = argsMap.has("r") ? -1 : 1;
  let nullsfirst = (argsMap.get("nulls") === "first");
  let limit = parseInt(argsMap.get("limit"));
  return { op: "SortProc", fields, sortdir, limit, nullsfirst };
}

function makeTopProc(fields, limit, flush) {
  if (limit === null) { limit = undefined; }
  if (fields === null) { fields = undefined; }
  flush = !!flush
  return { op: "TopProc", fields, limit, flush};
}

function makeCutProc(fields) { return { op: "CutProc", fields }; }
function makeHeadProc(count) { return { op: "HeadProc", count }; }
function makeTailProc(count) { return { op: "TailProc", count }; }
function makeUniqProc(cflag) { return { op: "TailProc", cflag }; }
function makeFilterProc(filter) { return { op: "FilterProc", filter }; }
function makePutProc(target, expression) { return { op: "PutProc", target, expression }; }
function makeReducer(op, var_, field) {
  if (field === null) { field = undefined; }
  return { op, var: var_, field };
}
function overrideReducerVar(reducer, v) {
  reducer.var = v;
  return reducer;
}

function makeDuration(seconds) {
  return {type: "Duration", seconds};
}

function makeReducerProc(reducers) {
  return { op: "ReducerProc", reducers };
}

function makeGroupByProc(duration, limit, keys, reducers) {
  if (limit === null) { limit = undefined; }
  return { op: "GroupByProc", keys, reducers, duration, limit };
}

function makeUnaryExpr(operator, operand) {
  return { op: "UnaryExpr", operator, operand };
}

function makeBinaryExprChain(first, rest) {
  let ret = first
  for (let part of rest) {
    ret = { op: "BinaryExpr", operator: part[1], lhs: ret, rhs: part[3] };
  }
  return ret
}

function makeConditionalExpr(condition, thenClause, elseClause) {
  return { op: "ConditionalExpr", condition, then: thenClause, else: elseClause };
}

function makeFunctionCall(fn, args) {
  return { op: "FunctionCall", function: fn, args };
}

function joinChars(chars) {
  return chars.join("");
}

function toLowerCase(str) {
  return str.toLowerCase();
}

function OR(a, b) {
  return a || b
}

function makeUnicodeChar(chars) {
  let n = parseInt(chars.join(""), 16);
  if (n < 0x10000) {
    return String.fromCharCode(n);
  }

  // stupid javascript 16 bit code points...
  n -= 0x10000;
  let surrogate1 = 0xD800 + ((n >> 10) & 0x7ff);
  let surrogate2 = 0xDC00 + (n & 0x3ff);
  return String.fromCharCode(surrogate1) + String.fromCharCode(surrogate2);
}

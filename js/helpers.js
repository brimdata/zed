function MakeSequentialProc (procs) {
  return { op: "SequentialProc", procs };
}

function MakeParallelProc (procs) {
  return { op: "ParallelProc", procs };
}

function MakeTypedValue (type, value) { return { type, value }; }
function GetValueType (v) { return v.type; }

function MakeFieldCall (fn, field, param) {
  return { op: "FieldCall", fn, field, param };
}
function ChainFieldCalls (base, derefs) {
  let ret = { op: "FieldRead", field: base };
  for (let d of derefs) {
    d.field = ret
    ret = d
  }
  return ret
}

function MakeBooleanLiteral (value) {
  return { op: "BooleanLiteral", value };
}

function MakeCompareField (comparator, field, value) {
  return { op: "CompareField", comparator, field, value };
}

function MakeCompareAny (comparator, recursive, value) {
  return { op: "CompareAny", comparator, recursive, value };
}

function MakeLogicalNot(expr) { return { op: "LogicalNot", expr }; }

function MakeChain(first, rest, op) {
  if (!rest || rest.length == 0) {
    return first;
  }
  let result = first;
  for (let term of rest) {
    result = { op, left: result, right: term };
  }
  return result;
}

function MakeOrChain(first, rest) {
  return MakeChain(first, rest, "LogicalOr");
}
function MakeAndChain(first, rest) {
  return MakeChain(first, rest, "LogicalAnd");
}

function MakeSearchString(value) {
  return { op: "SearchString", value };
}
function ResetSearchStringType(v) {
  v.type = "string";
}

function MakeSortProc(fields, sortdir, limit) {
  if (limit === null) { limit = undefined; }
  return { op: "SortProc", fields, sortdir, limit };
}

function MakeTopProc(fields, limit, flush) {
  if (limit === null) { limit = undefined; }
  if (fields === null) { fields = undefined; }
  flush = !!flush
  return { op: "TopProc", fields, limit, flush};
}

function MakeCutProc(fields)    { return { op: "CutProc", fields }; }
function MakeHeadProc(count)    { return { op: "HeadProc", count }; }
function MakeTailProc(count)    { return { op: "TailProc", count }; }
function MakeUniqProc(cflag)    { return { op: "TailProc", cflag }; }
function MakeFilterProc(filter) { return { op: "FilterProc", filter }; }

function MakeReducer(op, var_, field) {
  if (field === null) { field = undefined; }
  return { op, var: var_, field };
}
function OverrideReducerVar (reducer, v ) {
  reducer.var = v;
  return reducer;
}

function MakeDuration (seconds) { return {type: "Duration", seconds}; }
function MakeReducerProc (reducers) { return { op: "ReducerProc", reducers }; }

function MakeGroupByProc (duration, limit, keys, reducers) {
  if (limit === null) { limit = undefined; }
  return { op: "GroupByProc", keys, reducers, duration, limit };
}

function JoinChars (chars) { return chars.join(""); }
function ToLowerCase (str) { return str.toLowerCase(); }
function OR  (a, b) { return a || b }

module.exports = {
    MakeSequentialProc,
    MakeParallelProc,
    MakeTypedValue,
    GetValueType,
    MakeFieldCall,
    ChainFieldCalls,
    MakeBooleanLiteral,
    MakeCompareField, 
    MakeCompareAny,
    MakeLogicalNot,
    MakeChain,
    MakeOrChain,
    MakeAndChain,
    MakeSearchString,
    ResetSearchStringType,
    MakeSortProc,
	MakeTopProc,
	MakeCutProc,
	MakeTailProc,
	MakeUniqProc,
	MakeFilterProc,
	MakeReducer,
	OverrideReducerVar,
	MakeDuration,
	MakeReducerProc,
	MakeGroupByProc,
	JoinChars,
	ToLowerCase,
	OR,
	ParseInt: parseInt,
	ParseFloat: parseFloat,
}

let reglob = require("../../pkg/reglob/reglob")

function makeArgMap(args) {
  let m = {};
  for (let arg of args) {
    if (arg.name in m) {
      throw new Error(`Duplicate argument -${arg.name}`);
    }
    m[arg.name] = arg.value
  }
  return m
}

function makeBinaryExprChain(first, rest) {
  let ret = first
  for (let part of rest) {
    ret = { kind: "BinaryExpr", op: part[0], lhs: ret, rhs: part[1] };
  }
  return ret
}

function makeTemplateExprChain(args) {
  let ret = args[0]
  for (let part of args.slice(1)) {
    ret = { kind: "BinaryExpr", op: "+", lhs: ret, rhs: part };
  }
  return ret
}

function joinChars(chars) {
  return chars.join("");
}

function OR(a, b) {
  return a || b
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

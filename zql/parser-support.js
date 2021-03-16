let reglob = require("../reglob/reglob")

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

'use strict';

const fs = require('fs');

const zql = require('../zql')

function show(obj) {
    console.log(JSON.stringify(obj, null, 4));
}

function wrap(e) {
    return { op: 'Error', error: e };
}

function parse_query(line) {
    try {
        return zql.parse(line);
    } catch (e) {
        return wrap(e);
    }
}

function parse_cli(line) {
    try {
        return zql.parse(line);
    } catch (e) {
        // XXX do nothing
    }
    return undefined;
}

function parse(src) {
    let ast = parse_cli(src);
    if (!ast) {
        ast = parse_query(src);
    }
    return ast;
}

let filename = '/dev/stdin';
let argv = process.argv;
if (argv.length === 3) {
    filename = argv[2];
}

let boom_src;
try {
    boom_src = fs.readFileSync(filename, 'utf8');
} catch (e) {
    show(wrap(e));
    process.exit(1);
}
show(parse(boom_src));

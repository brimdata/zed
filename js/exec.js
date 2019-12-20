'use strict';

const zql = require("./zql.js")

function stringify(obj) {
	return JSON.stringify(obj, null, 4)
}

function wrap(e) {
    return { op: 'Error', error: e };
}

let argv = process.argv;
let boom_src;
if (argv.length >= 3) {
    boom_src = argv[2];
}
else {
    console.error(stringify(wrap("no query arg found")));
    process.exit(1);
    return 
}

try {
    console.log(stringify(zql.parse(boom_src)));
    process.exit(0)
} catch (e) {
    console.error(stringify(wrap(e)));
    process.exit(1)
}

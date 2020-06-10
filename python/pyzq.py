#!/usr/bin/env python3

import os
import sys

from cffi import FFI

ZQLIB = os.path.join(os.path.dirname(os.path.abspath(__file__)),
                     './build/zq.so')


class Zq(object):
    def __init__(self):
        self.ffi = FFI()
        self.ffi.cdef("""
void free(void *ptr);

typedef long GoInt;
typedef unsigned char GoUint8;
typedef struct { const char *p; GoInt n; } GoString;

struct goresult {
	char* r0;
	GoUint8 r1;
};

extern struct goresult ZqlFileEval(GoString p0, GoString p1, GoString p2);
extern struct goresult ErrorTest();
""")
        self.zqlib = self.ffi.dlopen(ZQLIB)

    def gostring(self, refs, s):
        """Convert a Python string to an allocated GoString."""
        u8b = s.encode('utf-8')
        c = self.ffi.new('char[]', u8b)
        refs.append(c)
        gstr = self.ffi.new('GoString*', {'p': c, 'n': len(u8b)})
        refs.append(gstr)
        return gstr

    def result(self, res):
        """Interpret a goresult structure, raising an exception for errors."""
        if res.r1 == 1:
            return
        err = str(self.ffi.string(res.r0), 'utf-8')
        self.zqlib.free(res.r0)
        raise Exception(err)

    def error_test(self):
        self.result(self.zqlib.ErrorTest())

    def zql_file_eval(self, zql, infile, outfile):
        refs = []
        gzql = self.gostring(refs, zql)
        ginfile = self.gostring(refs, infile)
        goutfile = self.gostring(refs, outfile)
        self.result(self.zqlib.ZqlFileEval(gzql[0], ginfile[0], goutfile[0]))


if __name__ == "__main__":
    if len(sys.argv) < 4:
        raise Exception("expected args: <zql> <input-file> <output-file>")

    zql = sys.argv[1]
    infile = sys.argv[2]
    outfile = sys.argv[3]

    zq = Zq()
    zq.zql_file_eval(zql, infile, outfile)

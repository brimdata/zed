import os
import sys

from cffi import FFI

# This ffibuilder code expects an archive via the Go c-archive buildmode:
# go build -buildmode=c-archive -o python/brim/build/zqext/libzqext.a python/brim/src/zqext.go
zqextbuild = os.path.abspath(
    os.path.join(os.path.dirname(os.path.abspath(__file__)),
                 '..', 'build', 'zqext'))

extra_link_args = []
if sys.platform == 'darwin':
    extra_link_args = ['-framework', 'Security']

ffibuilder = FFI()
ffibuilder.cdef("""
void free(void *ptr);

typedef unsigned char GoUint8;
typedef struct { const char *p; ptrdiff_t n; } GoString;

struct ErrorTest_return {
	char* r0;
	GoUint8 r1;
};

extern struct ErrorTest_return ErrorTest();

struct ZedFileEval_return {
	char* r0;
	GoUint8 r1;
};

extern struct ZedFileEval_return ZedFileEval(GoString p0, GoString p1, GoString p2, GoString p3, GoString p4);
""")

ffibuilder.set_source("_zqext",
                      """
                           #include "libzqext.h"
                      """,
                      libraries=['zqext'],
                      library_dirs=[zqextbuild],
                      include_dirs=[zqextbuild],
                      extra_link_args=extra_link_args)

if __name__ == "__main__":
    ffibuilder.compile(verbose=True)

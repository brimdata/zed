from brim._zqext import ffi, lib


def gostring(objrefs, s):
    """Convert a Python string to an allocated GoString. cdata objects are
    added to the objrefs list to ensure the caller controls when they go
    out of scope to prevent premature collection."""
    u8b = s.encode('utf-8')
    c = ffi.new('char[]', u8b)
    objrefs.append(c)
    gstr = ffi.new('GoString*', {'p': c, 'n': len(u8b)})
    objrefs.append(gstr)
    return gstr


def checkresult(res):
    """Interpret a goresult structure, raising an exception for errors."""
    if res.r1 == 1:
        return
    err = str(ffi.string(res.r0), 'utf-8')
    lib.free(res.r0)
    raise Exception(err)


def error_test():
    """Used to verify error passing from Go."""
    checkresult(lib.ErrorTest())


def zed_file_eval(inquery, infile, informat, outfile, outformat):
    # objrefs holds references to the allocated cdata objects for the
    # lifetime of the zed file call.
    objrefs = []
    gquery = gostring(objrefs, inquery)
    ginfile = gostring(objrefs, infile)
    ginformat = gostring(objrefs, informat)
    goutfile = gostring(objrefs, outfile)
    goutformat = gostring(objrefs, outformat)
    checkresult(lib.ZedFileEval(gquery[0], ginfile[0], ginformat[0], goutfile[0],
                                goutformat[0]))

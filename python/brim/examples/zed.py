import argparse
import sys

import brim

parser = argparse.ArgumentParser(description='Evaluate Zed query against a named file (or stdin by specifying "-")')
parser.add_argument('-i', metavar='input-format', default='auto', help='input data format [auto,zng,ndjson,zeek,zjson,tzng,parquet]')
parser.add_argument('-f', metavar='output-format', default='zng', help='output data format [zng,ndjson,table,text,types,zeek,zjson,tzng]')
parser.add_argument('args', nargs='*', help='<query>  [ <input-file> | - ]  [ <output-file> | - ]')
args = parser.parse_args()

if len(args.args) != 3:
    sys.exit(parser.print_help())

brim.zed_file_eval(
    inquery=args.args[0],
    infile=args.args[1],
    informat=args.i,
    outfile=args.args[2],
    outformat=args.f,
)

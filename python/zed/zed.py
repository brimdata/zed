import base64
import decimal
import getpass
import ipaddress
import json
import urllib.parse

import dateutil.parser
import durationpy
import requests


DEFAULT_BASE_URL = 'http://127.0.0.1:9867'


class Client():
    def __init__(self, base_url=DEFAULT_BASE_URL):
        self.base_url = base_url
        self.session = requests.Session()

    def create_pool(self, name, layout={'order': 'desc', 'keys': [['ts']]},
                    thresh=0):
        r = self.session.post(self.base_url + '/pool', json={
            'name': name,
            'layout': layout,
            'thresh': thresh,
        })
        r.raise_for_status()

    def load(self, pool_name_or_id, data, branch_name='main',
             commit_author=getpass.getuser(), commit_body=''):
        pool = urllib.parse.quote(pool_name_or_id)
        branch = urllib.parse.quote(branch_name)
        url = self.base_url + '/pool/' + pool + '/branch/' + branch
        commit_message = {'author': commit_author, 'body': commit_body}
        headers = {'Zed-Commit': json.dumps(commit_message)}
        r = self.session.post(url, headers=headers, data=data)
        r.raise_for_status()

    def query(self, query):
        return decode_raw(self.query_raw(query))

    def query_raw(self, query):
        body = {'query': query}
        r = self.session.post(self.base_url + '/query', json=body, stream=True)
        r.raise_for_status()
        return (json.loads(line) for line in r.iter_lines() if line)


def decode_raw(raw):
    types = {}
    for msg in raw:
        if msg['kind'] != 'Object':
            continue
        value = msg['value']
        if 'types' in value:
            for typ in value['types']:
                _decode_type(types, typ)
        yield _decode_value(types[value['schema']], value['values'])


def _decode_type(types, typ):
    kind = typ['kind']
    if kind == 'typedef':
        t = _decode_type(types, typ['type'])
        types[typ['name']] = t
        return t
    if kind == 'typename':
        return types[typ['name']]
    if kind == 'primitive':
        return typ
    if kind == 'record':
        for f in typ['fields']:
            f['type'] = _decode_type(types, f['type'])
        return typ
    if kind in ['array', 'set']:
        typ['type'] = _decode_type(types, typ['type'])
        return typ
    if kind == 'enum':
        raise 'unimplemented'
    if kind == 'map':
        typ['key_type'] = _decode_type(types, typ['key_type'])
        typ['val_type'] = _decode_type(types, typ['val_type'])
        return typ
    if kind == 'union':
        typ['types'] = [_decode_type(types, t) for t in typ['types']]
        return typ
    raise Exception(f'unknown type kind {kind}')


def _decode_value(typ, value):
    if value is None:
        return None
    kind = typ['kind']
    if kind == 'primitive':
        name = typ['name']
        if name in ['uint8', 'uint16', 'uint32', 'uint64',
                    'int8', 'int16', 'int32', 'int64']:
            return int(value)
        if name == 'duration':
            return durationpy.from_str(value)
        if name == 'time':
            return dateutil.parser.isoparse(value)
        if name in ['float16', 'float32', 'float64']:
            return float(value)
        if name == 'decimal':
            return decimal.Decimal(value)
        if name == 'bool':
            return value == 'T'
        if name == 'bytes':
            return base64.b64decode(value, validate=True)
        if name in ['string', 'bstring']:
            return value
        if name == 'ip':
            return ipaddress.ip_address(value)
        if name == 'net':
            return ipaddress.ip_network(value)
        if name in ['type', 'error']:
            return value
        if name == 'null':
            return None
        raise Exception(f'unknown primitive name {name}')
    if kind == 'record':
        return {f['name']: _decode_value(f['type'], v)
                for f, v in zip(typ['fields'], value)}
    if kind == 'array':
        return [_decode_value(typ['type'], v) for v in value]
    if kind == 'enum':
        raise 'unimplemented'
    if kind == 'map':
        key_type, val_type = typ['key_type'], typ['val_type']
        return {_decode_value(key_type, v[0]): _decode_value(val_type, v[1])
                for v in value}
    if kind == 'set':
        return {_decode_value(typ['type'], v) for v in value}
    if kind == 'union':
        type_index, val = value
        return _decode_value(typ['types'][int(type_index)], val)
    raise Exception(f'unknown type kind {kind}')


if __name__ == '__main__':
    import argparse
    import pprint

    parser = argparse.ArgumentParser(
        description='Send a query to zed and pretty-print results.',
        formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    parser.add_argument('-u', dest='base_url', default=DEFAULT_BASE_URL,
                        help='zed base URL')
    parser.add_argument('query')
    args = parser.parse_args()

    c = Client(args.base_url)
    for record in c.search(args.query):
        pprint.pprint(record)

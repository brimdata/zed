import base64
import datetime
import decimal
import ipaddress
import json
import requests


DEFAULT_BASE_URL = 'http://127.0.0.1:9867'


class Client():
    def __init__(self, base_url=DEFAULT_BASE_URL):
        self.base_url = base_url
        self.session = requests.Session()

    def ast(self, zql):
        r = self.session.post(self.base_url + "/ast", json={'zql': zql})
        r.raise_for_status()
        return r.json()

    def search(self, space_name, zql):
        return decode_raw(self.search_raw(space_name, zql))

    def search_raw(self, space_name, zql):
        body = {
            'dir': -1,
            'proc': self.ast(zql),
            'space': self.spaces()[space_name]['id'],
        }
        params = {'format': 'zjson'}
        r = self.session.post(self.base_url + "/search", json=body,
                              params=params, stream=True)
        r.raise_for_status()
        # Return rather than yield to raise exceptions earlier.
        return (json.loads(line) for line in r.iter_lines() if line)

    def spaces(self):
        r = self.session.get(self.base_url + "/space")
        r.raise_for_status()
        return {s['name']: s for s in r.json()}


def decode_raw(raw):
    types = {}
    for msg in raw:
        if msg['type'] != 'SearchRecords':
            continue
        for rec in msg['records']:
            if 'types' in rec:
                for typ in rec['types']:
                    _decode_type(types, typ)
            yield _decode_value(types[rec['schema']], rec['values'])


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
            return datetime.timedelta(seconds=float(value))
        if name == 'time':
            return datetime.datetime.utcfromtimestamp(float(value))
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
        description='Send a query to zqd and pretty-print results.',
        formatter_class=argparse.ArgumentDefaultsHelpFormatter)
    parser.add_argument('-u', dest='base_url', default=DEFAULT_BASE_URL,
                        help='zqd base URL')
    parser.add_argument('space_name')
    parser.add_argument('zql')
    args = parser.parse_args()

    c = Client(args.base_url)
    for record in c.search(args.space_name, args.zql):
        pprint.pprint(record)

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
        aliases, schemas = {}, {}
        for raw in self.search_raw(space_name, zql):
            if raw['type'] == 'SearchRecords':
                for rec in raw['records']:
                    if 'aliases' in rec:
                        for a in rec['aliases']:
                            aliases[a['name']] = a['type']
                    if 'schema' in rec:
                        schemas[rec['id']] = rec['schema']
                    yield _to_native(aliases, schemas[rec['id']], rec['values'])

    def search_raw(self, space_name, zql):
        body = {
            'dir': -1,
            'proc': self.ast(zql),
            'space': self.spaces()[space_name]['id'],
        }
        params = {'format': 'zjson'}
        r = self.session.post(self.base_url + "/search", json=body, params=params, stream=True)
        r.raise_for_status()
        for line in r.iter_lines():
            if line:
                yield json.loads(line)

    def spaces(self):
        r = self.session.get(self.base_url + "/space")
        r.raise_for_status()
        return {s['name']: s for s in r.json()}


def _to_native(aliases, schema, value):
    if value is None:
        return None
    typ = schema['type']
    typ = aliases.get(typ, typ)
    if typ == 'record':
        return {of['name']: _to_native(aliases, of, v) for of, v in zip(schema['of'], value)}
    if typ in ['array', 'set']:
        of = schema['of']
        if type(of) is str:
            of = {'type': of}
        return [_to_native(aliases, of, v) for v in value]
    if typ in ['enum', 'union']:
        raise 'unimplemented'
    if typ in ['uint8', 'uint16', 'uint32', 'uint64', 'int8', 'int16', 'int32', 'int64']:
        return int(value)
    if typ == 'duration':
        return datetime.timedelta(seconds=float(value))
    if typ == 'time':
        return datetime.datetime.utcfromtimestamp(float(value))
    if typ in ['float16', 'float32', 'float64']:
        return float(value)
    if typ == 'decimal':
        return decimal.Decimal(value)
    if typ == 'bool':
        return value == 'T'
    if typ in ['string', 'bstring']:
        return str(value)
    if typ == 'ip':
        return ipaddress.ip_address(value)
    if typ == 'net':
        return ipaddress.ip_network(value)
    if typ == ['type', 'error']:
        return str(value)
    if typ == 'null':
        return None
    raise Exception('type {} is unknown'.format(typ))


if __name__ == '__main__':
    import argparse
    import pprint

    parser = argparse.ArgumentParser(description='Search zqd.')
    parser.add_argument('-u', dest='base_url', default=DEFAULT_BASE_URL, help='base URL')
    parser.add_argument('space_name')
    parser.add_argument('zql')
    args = parser.parse_args()

    c = Client(args.base_url)
    for rec in c.search(args.space_name, args.zql):
        pprint.pprint(rec)

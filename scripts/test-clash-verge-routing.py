#!/usr/bin/env python3
"""Isolated Mihomo routing/provider integration check (Python 3 + PyYAML).

Usage: python scripts/test-clash-verge-routing.py MIHOMO CONFIG PROVIDER_CACHE_DIR
CONFIG is emitted by go run ./cmd/verify-clash-profile. PROVIDER_CACHE_DIR holds
its four downloaded MRS paths. No traffic reaches the test destinations:
DIRECT actions and the PROXY group use local HTTP CONNECT capture adapters.
Providers serve the actual downloaded bytes over local HTTP with a one-second
interval, so initial loads and refreshes can be checked without a 24-hour wait.
"""
import collections
import http.server
import json
import pathlib
import socket
import socketserver
import struct
import subprocess
import sys
import tempfile
import threading
import time
import urllib.request

import yaml

binary, config_path, cache_path = map(pathlib.Path, sys.argv[1:4])
base = yaml.safe_load(config_path.read_text())
payloads = {name: (cache_path / p['path']).read_bytes()
            for name, p in base['rule-providers'].items()}
requests = collections.Counter()
captures = []
lock = threading.Lock()
stop = threading.Event()


class LocalServer(http.server.BaseHTTPRequestHandler):
    def log_message(self, *_):
        pass

    def do_GET(self):
        name = self.path.lstrip('/')
        if name not in payloads:
            self.send_error(404)
            return
        with lock:
            requests[name] += 1
        self.send_response(200)
        self.send_header('Content-Length', str(len(payloads[name])))
        self.end_headers()
        self.wfile.write(payloads[name])

    def do_CONNECT(self):
        with lock:
            captures.append(self.path)
        self.send_response(200)
        self.end_headers()
        self.wfile.flush()
        stop.wait(15)


def free_port():
    with socket.socket() as sock:
        sock.bind(('127.0.0.1', 0))
        return sock.getsockname()[1]


dns_records = {'foreign.fixture.example.com': '8.8.8.8',
               'unknown.internal.example.com': '192.168.4.10',
               'unlisted-cn.fixture.example.com': '223.5.5.5'}


class LocalDNS(socketserver.BaseRequestHandler):
    def handle(self):
        packet, sock = self.request
        offset, labels = 12, []
        while packet[offset]:
            size = packet[offset]
            labels.append(packet[offset + 1:offset + 1 + size].decode())
            offset += size + 1
        offset += 1
        qtype, qclass = struct.unpack('!HH', packet[offset:offset + 4])
        question = packet[12:offset + 4]
        address = dns_records.get('.'.join(labels))
        answer = b''
        if address and qtype == 1:
            answer = b'\xc0\x0c' + struct.pack('!HHIH', 1, 1, 60, 4) + socket.inet_aton(address)
        header = packet[:2] + struct.pack('!HHHHH', 0x8180, 1, int(bool(answer)), 0, 0)
        sock.sendto(header + question + answer, self.client_address)


dns_server = socketserver.ThreadingUDPServer(('127.0.0.1', 0), LocalDNS)
dns_server.daemon_threads = True
threading.Thread(target=dns_server.serve_forever, daemon=True).start()
server = http.server.ThreadingHTTPServer(('127.0.0.1', 0), LocalServer)
server.daemon_threads = True
threading.Thread(target=server.serve_forever, daemon=True).start()
port = server.server_port
api_port, mixed_port = free_port(), free_port()
opener = urllib.request.build_opener(urllib.request.ProxyHandler({}))


def get_api(path):
    return json.load(opener.open(f'http://127.0.0.1:{api_port}{path}', timeout=2))


def until(fn, timeout=15):
    end = time.monotonic() + timeout
    last = None
    while time.monotonic() < end:
        try:
            last = fn()
            if last:
                return last
        except (OSError, ValueError):
            pass
        time.sleep(0.05)
    raise AssertionError(f'timed out, last result: {last}')


clients = []
with tempfile.TemporaryDirectory(prefix='mihomo-routing-') as temp:
    home = pathlib.Path(temp)
    config = dict(base)
    config['proxies'] = [dict(name=name, type='http', server='127.0.0.1', port=port)
                         for name in ['TEST-DIRECT', 'TEST-PROXY']]
    config['proxy-groups'] = [dict(name='PROXY', type='select', proxies=['TEST-PROXY'])]
    config['rules'] = [rule.replace(',DIRECT', ',TEST-DIRECT') for rule in base['rules']]
    config['external-controller'] = f'127.0.0.1:{api_port}'
    config['mixed-port'] = mixed_port
    config['allow-lan'] = False
    config['ipv6'] = True
    config['find-process-mode'] = 'off'
    config['dns'] = dict(enable=True, ipv6=True, nameserver=[f'udp://127.0.0.1:{dns_server.server_address[1]}'])
    for name, provider in config['rule-providers'].items():
        provider['url'] = f'http://127.0.0.1:{port}/{name}'
        provider['proxy'] = 'DIRECT'
        provider['interval'] = 1
    (home / 'config.yaml').write_text(yaml.safe_dump(config, allow_unicode=True))
    with (home / 'runtime.log').open('w+') as log:
        process = subprocess.Popen([str(binary.resolve()), '-d', temp, '-f', str(home / 'config.yaml')],
                                   stdout=log, stderr=subprocess.STDOUT)
        try:
            until(lambda: get_api('/version'))
            providers = until(lambda: (p if all(v['ruleCount'] > 0 for v in p.values()) else None)
                              if (p := get_api('/providers/rules')['providers']) else None)
            tests = [
                ('192.168.1.10', 'TEST-DIRECT', 'IPCIDR'),
                ('10.8.0.1', 'TEST-DIRECT', 'IPCIDR'),
                ('172.16.1.2', 'TEST-DIRECT', 'IPCIDR'),
                ('100.64.1.2', 'TEST-DIRECT', 'IPCIDR'),
                ('169.254.1.2', 'TEST-DIRECT', 'IPCIDR'),
                ('127.0.0.1', 'TEST-DIRECT', 'IPCIDR'),
                ('fd00::1', 'TEST-DIRECT', 'IPCIDR'),
                ('fe80::1', 'TEST-DIRECT', 'IPCIDR'),
                ('::1', 'TEST-DIRECT', 'IPCIDR'),
                ('router.lan', 'TEST-DIRECT', 'DomainSuffix'),
                ('nas.local', 'TEST-DIRECT', 'DomainSuffix'),
                ('printer.home.arpa', 'TEST-DIRECT', 'DomainSuffix'),
                ('baidu.com', 'TEST-DIRECT', 'RuleSet'),
                ('223.5.5.5', 'TEST-DIRECT', 'RuleSet'),
                ('240e::1', 'TEST-DIRECT', 'RuleSet'),
                ('unknown.internal.example.com', 'TEST-DIRECT', 'RuleSet'),
                ('unlisted-cn.fixture.example.com', 'TEST-DIRECT', 'RuleSet'),
                ('8.8.8.8', 'TEST-PROXY', 'Match'),
                ('2001:4860:4860::8888', 'TEST-PROXY', 'Match'),
                ('foreign.fixture.example.com', 'TEST-PROXY', 'Match'),
            ]
            results = []
            for host, expected_adapter, expected_rule in tests:
                sock = socket.create_connection(('127.0.0.1', mixed_port), timeout=3)
                clients.append(sock)
                source_port = str(sock.getsockname()[1])
                authority = f'[{host}]:443' if ':' in host else f'{host}:443'
                sock.sendall(f'CONNECT {authority} HTTP/1.1\r\nHost: {authority}\r\n\r\n'.encode())
                response = sock.recv(4096)
                assert b'200' in response.split(b'\r\n')[0], (host, response)
                connection = until(lambda: next((c for c in (get_api('/connections')['connections'] or [])
                    if str(c['metadata']['sourcePort']) == source_port), None))
                assert expected_adapter in connection['chains'], (host, connection)
                assert connection['rule'].lower() == expected_rule.lower(), (host, connection)
                results.append(dict(destination=host, adapter=expected_adapter,
                                    rule=connection['rule'], payload=connection['rulePayload']))
            until(lambda: all(count >= 2 for count in requests.values()) and len(requests) == 4)
            report = dict(core=get_api('/version'), providers={name: p['ruleCount'] for name, p in providers.items()},
                          provider_http_requests=dict(requests), routing_cases=results)
            print(json.dumps(report, indent=2))
        except Exception:
            log.flush()
            log.seek(0)
            print(log.read(), file=sys.stderr)
            raise
        finally:
            for sock in clients:
                sock.close()
            process.terminate()
            process.wait(timeout=10)
            stop.set()
            server.shutdown()
            dns_server.shutdown()

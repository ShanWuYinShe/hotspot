#!/usr/bin/env python3
import argparse
import json
import sys
import urllib.error
import urllib.request
from pathlib import Path
from urllib.parse import urljoin


def fetch(url: str, method: str = 'GET') -> dict:
    req = urllib.request.Request(url, method=method)
    try:
        with urllib.request.urlopen(req, timeout=15) as resp:
            body = resp.read()
            return {
                'url': url,
                'method': method,
                'status': resp.status,
                'headers': dict(resp.headers.items()),
                'body_size': len(body),
            }
    except urllib.error.HTTPError as e:
        body = e.read()
        return {
            'url': url,
            'method': method,
            'status': e.code,
            'headers': dict(e.headers.items()),
            'body_size': len(body),
        }


def main() -> int:
    parser = argparse.ArgumentParser(description='采集 verify-inspect 页面级巡检证据；脚本直接基于当前 worktree 执行，不需要额外 cd')
    parser.add_argument('--base-url', default='http://localhost:38088')
    parser.add_argument('--log-dir', required=True)
    parser.add_argument('--routes', nargs='*', default=['/', '/auth', '/settings'])
    args = parser.parse_args()

    log_dir = Path(args.log_dir)
    log_dir.mkdir(parents=True, exist_ok=True)

    report = {
        'base_url': args.base_url,
        'checks': []
    }

    for route in args.routes:
        report['checks'].append(fetch(urljoin(args.base_url, route), method='HEAD'))

    report['checks'].append(fetch(urljoin(args.base_url, '/api/v1/system/health'), method='HEAD'))

    out = log_dir / 'inspect-evidence.json'
    out.write_text(json.dumps(report, ensure_ascii=False, indent=2) + '\n', encoding='utf-8')
    print(str(out))
    return 0


if __name__ == '__main__':
    raise SystemExit(main())

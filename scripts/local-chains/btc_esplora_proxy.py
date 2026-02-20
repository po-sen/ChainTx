#!/usr/bin/env python3

import json
import os
from decimal import Decimal
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import unquote
from urllib.request import Request, urlopen


BTC_RPC_URL = os.getenv("BTC_RPC_URL", "http://127.0.0.1:18443")
BTC_RPC_USER = os.getenv("BTC_RPC_USER", "chaintx")
BTC_RPC_PASSWORD = os.getenv("BTC_RPC_PASSWORD", "chaintx")
PORT = int(os.getenv("BTC_ESPLORA_PROXY_PORT", "3999"))


def rpc_call(method: str, params):
    payload = json.dumps(
        {
            "jsonrpc": "1.0",
            "id": "btc-esplora-proxy",
            "method": method,
            "params": params,
        }
    ).encode("utf-8")

    req = Request(BTC_RPC_URL, data=payload, method="POST")
    auth = f"{BTC_RPC_USER}:{BTC_RPC_PASSWORD}".encode("utf-8")
    req.add_header("Authorization", "Basic " + __import__("base64").b64encode(auth).decode("ascii"))
    req.add_header("Content-Type", "application/json")

    with urlopen(req, timeout=5) as resp:
        body = json.loads(resp.read().decode("utf-8"))
    if body.get("error") is not None:
        raise RuntimeError(str(body["error"]))
    return body.get("result")


def confirmed_sats_for_address(address: str) -> int:
    descriptor = f"addr({address})"
    result = rpc_call("scantxoutset", ["start", [descriptor]])
    total_amount = Decimal(str(result.get("total_amount", "0")))
    sats = int(total_amount * Decimal("100000000"))
    if sats < 0:
        return 0
    return sats


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.startswith("/address/"):
            address = unquote(self.path[len("/address/") :]).strip()
            if not address:
                self.send_error(400, "missing address")
                return
            try:
                sats = confirmed_sats_for_address(address)
            except Exception as exc:
                self.send_response(500)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(json.dumps({"error": str(exc)}).encode("utf-8"))
                return

            payload = {
                "chain_stats": {"funded_txo_sum": sats},
                "mempool_stats": {"funded_txo_sum": 0},
            }
            encoded = json.dumps(payload).encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)
            return

        self.send_error(404, "not found")

    def log_message(self, fmt, *args):
        return


def main():
    server = HTTPServer(("127.0.0.1", PORT), Handler)
    print(f"btc esplora proxy listening on 127.0.0.1:{PORT}", flush=True)
    server.serve_forever()


if __name__ == "__main__":
    main()

#!/usr/bin/env python3

import json
import os
from decimal import Decimal
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.error import HTTPError
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

    try:
        with urlopen(req, timeout=5) as resp:
            body = json.loads(resp.read().decode("utf-8"))
    except HTTPError as err:
        raw = err.read().decode("utf-8", errors="replace")
        raise RuntimeError(f"rpc http {err.code}: {raw}") from err
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


def tip_height() -> int:
    return int(rpc_call("getblockcount", []))


def utxos_for_address(address: str):
    descriptor = f"addr({address})"
    result = rpc_call("scantxoutset", ["start", [descriptor]]) or {}
    unspents = result.get("unspents", []) or []

    out = []
    for row in unspents:
        value_btc = Decimal(str(row.get("amount", "0")))
        value_sats = int(value_btc * Decimal("100000000"))
        block_height = int(row.get("height", 0) or 0)
        out.append(
            {
                "txid": row.get("txid"),
                "vout": row.get("vout"),
                "value": value_sats,
                "status": {
                    "confirmed": block_height > 0,
                    "block_height": block_height if block_height > 0 else None,
                },
            }
        )

    return out


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path == "/blocks/tip/height":
            try:
                height = tip_height()
            except Exception as exc:
                self.send_response(500)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(json.dumps({"error": str(exc)}).encode("utf-8"))
                return

            encoded = str(height).encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)
            return

        if self.path.startswith("/address/") and self.path.endswith("/utxo"):
            raw = self.path[len("/address/") : -len("/utxo")]
            address = unquote(raw).strip("/")
            if not address:
                self.send_error(400, "missing address")
                return
            try:
                utxos = utxos_for_address(address)
            except Exception as exc:
                self.send_response(500)
                self.send_header("Content-Type", "application/json")
                self.end_headers()
                self.wfile.write(json.dumps({"error": str(exc)}).encode("utf-8"))
                return

            encoded = json.dumps(utxos).encode("utf-8")
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(encoded)))
            self.end_headers()
            self.wfile.write(encoded)
            return

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

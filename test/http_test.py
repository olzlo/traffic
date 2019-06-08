#!/usr/bin/python3
import http.server
import socketserver


port = 8080

with socketserver.TCPServer(("", port), http.server.SimpleHTTPRequestHandler) as httpd:
    print("serving at port", port)
    httpd.serve_forever()

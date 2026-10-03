"""Dessau's model-server launcher.

Runs the pinned mlx-lm's own server -- its argument parser, its model
provider, its response generator and its request handler -- on a Unix domain
socket instead of a TCP port, so that only processes running as the serving
account can reach it (iss-2610030846581757). The socket lives in a directory
only that account can open; this script refuses any other.

It is Dessau's own code, carried in the Dessau binary and handed to the
interpreter with -c under -I (internal/runtime/launcher.go launchArgs), so no
module from the working directory, the model directory or a PYTHONPATH is
imported in its place or in mlx_lm's.

Usage: python -I -u -c <this> --dessau-socket PATH [mlx_lm.server flags...]

What it adds to the server, as defence behind the gateway's own checks:

  * the request's "model" is always the model the server was launched with,
    whatever the request named, so a request cannot make it load another
    directory;
  * a request that carries "draft_model" or "adapters" -- the two fields the
    server hands to load() -- is refused, by the key's presence whatever its
    value, as the gateway refuses them;
  * a launch naming a draft model, an adapter or remote tokenizer code is
    refused, and no origin is allowed cross-origin access.
"""

import io
import json
import logging
import os
import socketserver
import stat
import sys

SOCKET_FLAG = "--dessau-socket"

# The fields the pinned server reads as an instruction to load something the
# request names. Kept in step with internal/gateway's loadFields.
REFUSED_FIELDS = (
    ("draft_model", '"draft_model" is not accepted: Dessau does not load a second model for a request'),
    ("adapters", '"adapters" is not accepted: Dessau does not load adapter weights a request names'),
)


def _fail(message):
    sys.stderr.write("dessau-serve: " + message + "\n")
    sys.stderr.flush()
    sys.exit(2)


def _take_socket(argv):
    if len(argv) < 2 or argv[0] != SOCKET_FLAG:
        _fail("the first argument must be " + SOCKET_FLAG + " PATH")
    return argv[1], argv[2:]


def _check_private_dir(path):
    """Refuse a socket whose directory is not this account's alone."""
    directory = os.path.dirname(path)
    try:
        st = os.lstat(directory)
    except OSError as e:
        _fail("socket directory cannot be read: %s" % e.strerror)
    if not stat.S_ISDIR(st.st_mode):
        _fail("socket directory is not a directory")
    if st.st_uid != os.geteuid():
        _fail("socket directory belongs to another account")
    if stat.S_IMODE(st.st_mode) != 0o700:
        _fail("socket directory is open to other accounts (mode %o)" % stat.S_IMODE(st.st_mode))


def _clear_stale(path):
    try:
        st = os.lstat(path)
    except FileNotFoundError:
        return
    if not stat.S_ISSOCK(st.st_mode):
        _fail("something other than a socket is in the socket's place")
    os.unlink(path)


class _UnixHTTPServer(socketserver.ThreadingUnixStreamServer):
    # As http.server.ThreadingHTTPServer: a request thread never holds the
    # process up at exit.
    daemon_threads = True


def _handler_class(base, model):
    class Handler(base):
        def address_string(self):
            # A Unix socket's peer has no (host, port); the base class would
            # index into an empty address to log each request.
            return "local"

        def _refuse(self, message):
            body = json.dumps(
                {"error": {"message": message, "type": "invalid_request_error", "code": 400}}
            ).encode()
            self.send_response(400)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)

        def do_POST(self):
            try:
                length = int(self.headers.get("Content-Length"))
            except (TypeError, ValueError):
                return super().do_POST()  # the server answers it as it does
            if length < 0:
                # The server would take it as "read to the end of the stream"
                # and parse a body this check never saw.
                self._refuse("Invalid Content-Length header")
                return
            raw = self.rfile.read(length)
            try:
                body = json.loads(raw.decode())
            except (UnicodeDecodeError, ValueError):
                body = None
            if isinstance(body, dict):
                for name, message in REFUSED_FIELDS:
                    if name in body:
                        self._refuse(message)
                        return
                body["model"] = model
                raw = json.dumps(body).encode()
            # Hand the server the body it is to read, then put the connection
            # back for whatever follows on it.
            connection = self.rfile
            self.rfile = io.BytesIO(raw)
            del self.headers["Content-Length"]
            self.headers["Content-Length"] = str(len(raw))
            try:
                super().do_POST()
            finally:
                self.rfile = connection

    return Handler


def _serve(server, path, model_provider):
    cli = model_provider.cli_args
    if getattr(cli, "draft_model", None) is not None:
        _fail("a draft model is not loaded")
    if getattr(cli, "adapter_path", None) is not None:
        _fail("adapter weights are not loaded")
    if getattr(cli, "trust_remote_code", False):
        _fail("remote tokenizer code is not trusted")
    if not cli.model:
        _fail("no model to serve")
    cli.allowed_origins = []

    prompt_cache = server.LRUPromptCache(cli.prompt_cache_size)
    generator = server.ResponseGenerator(model_provider, prompt_cache)
    handler = _handler_class(server.APIHandler, cli.model)
    fingerprint = server.get_system_fingerprint()

    _check_private_dir(path)
    _clear_stale(path)
    os.umask(0o077)
    httpd = _UnixHTTPServer(
        path,
        lambda *args, **kwargs: handler(generator, *args, system_fingerprint=fingerprint, **kwargs),
    )
    logging.info("Starting httpd on a private socket")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        httpd.shutdown()
        generator.stop_and_join()
    finally:
        try:
            os.unlink(path)
        except OSError:
            pass


def main():
    path, rest = _take_socket(sys.argv[1:])
    _check_private_dir(path)

    from mlx_lm import server

    def run(host, port, model_provider, server_class=None, handler_class=None):
        # The server's main() parses its flags and builds its model provider,
        # then calls this in place of its own run(), which would bind TCP.
        _serve(server, path, model_provider)

    server.run = run
    sys.argv = ["mlx_lm.server"] + rest
    server.main()


main()

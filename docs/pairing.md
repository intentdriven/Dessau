# Pair a chat client with your server

Pairing gives one chat client an identity of its own on your server. It makes a
key that never leaves the device it is on, your server signs it a certificate,
and from then on that client proves itself with the key on every request
instead of sending the API key. You see it on the control panel under
**Clients**, with the name it chose and when it was last heard from, and you
can revoke it with one click.

What pairing protects against, and what it does not, is on
[What pairing is worth](pairing-explained.md). Read that page before you rely
on this one.

Other clients are unaffected. curl, a Python script, anything that speaks the
OpenAI API goes on using the ordinary port and the shared API key exactly as it
does today.

## Pair a client

1. Open the chat client and point it at your server — pick it from the model
   picker, or type its address in **Settings**.
2. In **Settings**, under **Pairing**, give this device a name. Something you
   will recognise on a list: "Bob's iPad", "the kitchen Mac".
3. Press **Pair with this server**.

The client shows two fingerprints once it has paired: the server's key and its
own.

## Check what you paired with

Anything on your network can pair with your server, and anything on your
network can answer a pairing request. So the fingerprints are worth one look.

1. Open the control panel and go to **Clients**.
2. Compare **This server's key** with the server fingerprint the client shows
   in its **Pairing** settings. They match, or the client is talking to
   something that is not your server.
3. Find the client in the list below. Its own fingerprint is shown beside its
   name, which is how you tell two devices apart when they chose the same name.

## Revoke a client

1. Open the control panel and go to **Clients**.
2. Press **Revoke** beside the client, then press it again to confirm.

The next request that client makes is refused, including on a connection it
already had open. To use the server again it pairs again, which puts it back on
your list under a new row.

## Choose the port

Paired clients use a second port, one above the ordinary one by default. Set it
in **Settings**, under **Port for paired clients**:

- `0` — the port just above the ordinary one. This is the default.
- any port number — that port.
- `-1` — no port for paired clients at all. Nothing can pair, and anything
  already paired stops being able to connect.

Changing it takes effect when Gropius next starts.

## Start again

Your server keeps its key in `server-key.pem`, beside `config.json`. It survives
restarts, which is what lets every paired client go on working when you rename
the Mac or it changes address.

Deleting that file makes a new key, and every paired client stops recognising
your server. They each pair again, and your **Clients** list fills with rows you
can revoke.

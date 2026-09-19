# What pairing is worth

Pairing solves two problems the shared API key cannot, and leaves a third open.
This page says which is which, so you can decide what to rely on.

## The problem it solves

One API key is the same string on every device. You type it into each of them
by hand, it travels over your network with every request, and the only way to
take it away from one device is to change it for all of them. Your server
cannot tell one client from another, so it cannot tell you which of your
devices is busy, and it cannot tell you whether anything else has the key.

A paired client holds a key of its own. The private half is made on the device
and never leaves it, so there is nothing to read off one screen and type into
another. Your server knows each client by name, records when it last heard from
each one, and can revoke one without touching the others. What passes between a
paired client and your server is no longer readable by everything else on the
network.

## What it does not solve

**Anything on your network can pair.** Your server does not ask you first — that
is deliberate, so that pairing a device is one name and one press rather than a
code read off one screen and typed into another. The cost is that somebody else
on your network can pair a client of their own, and the only thing that tells
you so is reading the **Clients** list. Read it after you pair something, and
revoke anything you do not recognise.

**The first connection is the one you have to check.** A client learns your
server's key the first time it connects, and from then on refuses anything
else. If something on your network answers in your server's place at that exact
moment, the client learns that thing's key instead — and afterwards everything
looks right, including the lock beside the server's name. Nothing detects this.
Comparing the fingerprint the client shows against the one on your control panel
is what catches it, and it is the only thing that does.

**The ordinary port is still there.** Pairing adds a way in; it takes none away.
Anything holding the shared API key still reaches your server on the ordinary
port, and pairing does not change that. If the key has been shared more widely
than you meant, pairing a client does not help — change the key.

**A compromised device is compromised.** A key in a device's keychain is
available to whoever controls the device. Pairing says which device is talking,
never who is holding it.

**The key is what is trusted, not the certificate.** Your server checks the key
a client presents against the list on your panel, and looks at nothing else in
the certificate — so a client holding a paired key is let in under a
certificate it wrote itself, with whatever name it likes in it. That is how it
should be: what proves a client is the key, and the certificate is only the
wrapper the key travels in. What it means in practice is that a key is the
whole of an identity here. If one ever escapes the device it was made on,
revoking it on the panel is the only thing that helps, and nothing about the
certificate would have slowed it down.

**Your server is not a certificate authority.** It signs a certificate for each
client it pairs, and nothing else trusts those certificates or should. There is
no revocation list, no expiry to watch, and nothing to install anywhere: your
server checks each client against the list you can see on the panel, and that
list is the whole of the rule.

## What travels where

The pairing exchange itself is on the ordinary port. What crosses it is the
client's **public** key going out and its certificate coming back — neither is a
secret, and neither is worth anything to somebody reading them, because the
private half stays on the device. What matters about that exchange is not who
can read it but who can answer it, which is the two limits above.

Your server announces itself over Bonjour with its fingerprint attached, so a
client can tell one server from another and notice a key that changed. That
announcement is not evidence of anything: any device on the link can make one.
It is a convenience, and the fingerprint on your control panel is the copy to
trust.

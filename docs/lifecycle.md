# Install, repair and remove Dessau

Dessau puts itself on this Mac, repairs itself, takes itself away, and says
what is wrong with it. This page walks through each of those four tasks. For
the verbs, their flags, the exit codes and the machine-readable output, see
[Reference: the lifecycle verbs](lifecycle-reference.md).

## Install it

One line, in a terminal:

```sh
curl -fsSL https://raw.githubusercontent.com/intentdriven/Dessau/main/install.sh | bash
```

That command is a bootstrap and only a bootstrap: it does the part that has to
happen before a Dessau binary exists on this Mac. It downloads the current
release, verifies it against the checksums published beside it, clears the
quarantine attribute, and hands over to `dessau install` **inside the bundle
it has just verified** — never to a copy already on the Mac. The script and the
binary it calls are the same build, which is what makes the two halves of an
install one thing.

From there the binary does the work, in this order:

1. **Places the application.** `/Applications` when your account can write
   there, `~/Applications` when it cannot. The move is staged: the installed
   bundle is set aside in your own account's folder, the new one is put in its
   place, and the set-aside copy goes only once the new one is there. No
   failure leaves this Mac without an application — and where the set-aside
   copy cannot be put back, it waits where only your account can remove it and
   the command says where to find it.
2. **Asks once for the firewall grant**, through the system authorisation
   panel, with the reason on the panel.
3. **Installs the private Python and MLX runtime**, in the foreground, naming
   the stage and how many of the three stages are done. This is the part that
   takes minutes.
4. **Links the `dessau` command** into `~/.local/bin`, and says so. Nothing is
   elevated for this.
5. **Opens the application** and waits for it to answer, then says whether it
   is serving.

When the command returns, Dessau is in the menu bar. Click its icon for the
control panel, and carry on at
[Getting started](getting-started.md#3-download-a-model).

### The one administrator panel

Exactly one panel appears in an install, and it is for the macOS Application
Firewall. Without that entry, Dessau accepts a connection from another machine
and then drops it, so the LAN sees an empty response while `localhost` on this
Mac works.

The panel rather than a password prompt in the terminal, for two reasons. It
asks for **an administrator's name and password**, so your own account does not
have to be an administrator — someone else can type theirs. And a terminal
prompt cannot be answered under `curl … | bash` at all: there the script's own
remaining text is standard input. No lifecycle verb reads standard input for
any purpose.

**The panel appears again on every update.** These builds are ad-hoc signed,
with no Developer ID, so the code identity the firewall keys its entry to
changes with each build. An entry made for one build does not carry over to the
next, and the grant is made again.

Decline the panel and the install carries on to the end. It prints a warning
and the two commands that make the grant by hand, and the server answers on
this Mac while other machines see nothing.

### When `~/.local/bin` is not on the search path

The `dessau` command is a link in your own bin directory, which is not on
every Mac's search path. When it is not on yours, the install says so and
prints the line that fixes it. Add it to `~/.zshrc` (or `~/.bash_profile`):

```sh
export PATH="$HOME/.local/bin:$PATH"
```

Open a new terminal after saving, and `dessau status` answers. Until then the
command works by its full path, `~/.local/bin/dessau`.

### When your account is not an administrator

`/Applications` holds one bundle that every account on the Mac launches, and a
standard account cannot write there. The install puts the bundle in your own
`~/Applications` instead, which macOS indexes the same way, and keys the
firewall entry to that path. Installing for every account when one account
asked is not what was asked, so nothing is elevated to reach the machine-wide
directory.

## Repair an installation

Run the install verb again:

```sh
dessau install
```

It repairs what is missing rather than reinstalling what is not, and says which
of the two it did: either that the runtime was already complete and nothing was
reinstalled, or which pieces it put back. A complete installation takes seconds.

Repair acts on the bundle that is already installed. On a Mac with no Dessau
at either destination there is nothing to repair, and the verb refuses and
names the bootstrap — before it asks for anything, so a Mac with nothing
installed never raises an authorisation panel.

## When a stage fails

A stage that fails stops the run. The command exits non-zero, names the stage
and why it failed, and names the command that retries it:

```
dessau install: the MLX runtime failed: …
Retry with: dessau install
```

Retrying resumes rather than restarting: the stages already done are the ones
the repair above leaves alone.

Two things are warnings rather than failures, because the install is usable
without them. A firewall grant that was declined or could not be made prints
the two commands that make it by hand. An application that could not be opened
leaves everything installed.

The terminal's verdict is what was true when the command exited, and it is not
revisited. Dessau checks its own runtime whenever it starts, so a provisioning
run that failed in the terminal may well be finished by the app itself on the
next launch — the control panel is where that shows.

## Update it

```sh
dessau update
```

It fetches the current release, checks it against the checksums published
beside it, and puts it in place with the same staged swap the install uses.
Then it tells you two things rather than one:

```
installed: 0.5.0, at /Applications/DessauServer.app
serving:   0.4.0, on port 11535
```

Those are separate facts, and on some Macs they differ.

The serving line is read from the running server, which publishes which build
it is. Where it cannot be read — nothing is serving, another account holds the
port, or the server is a build older than the field — the line says that it
cannot be determined and says why. The version just installed is never printed
in its place, so the command never tells you your Mac is serving something it
may not be.

The command reports five things every time:

1. **The version it installed** — read by asking the downloaded build what it
   is, not by trusting the release it came from.
2. **The version this Mac is serving**, or that it cannot be determined. It is
   never the version just installed standing in for an answer.
3. **That the download was verified against the checksums published with the
   release.** That is the whole of the check. The bundles carry no Developer ID
   and Gatekeeper sees nothing.
4. **That the firewall grant was re-made, and why it has to be.** The
   administrator panel appears on every update — see below.
5. **That there is no way back.** Only the current release is published, so the
   previous release cannot be fetched.

### When someone else is logged in

On a Mac with fast user switching, the server port belongs to whichever session
started Dessau first. If that is not yours, the update still replaces the
bundle — and then says so plainly rather than reporting success:

```
This Mac is serving a version this command did not install.
The bundle just placed takes effect when that session logs out, or when
Dessau is restarted there.
Quitting it is not something this command can do: a quit request reaches only
this login session.
```

It does not quit the other session's server. It cannot — a quit request reaches
only the session that sent it — and it says so rather than trying. Other
accounts are counted, never named, so the output is safe to paste into a
message.

Where your copy is in `~/Applications` rather than `/Applications`, the report
separates the two: your own copy was updated, and the copy this Mac serves from
is somebody else's.

### When something unidentified holds the port

If the process on the server port answers Dessau's identity challenge and
answers it wrongly, the update stops before it downloads anything. It names the
port, says what it found, and leaves the installed application exactly as it
was. A Mac with an impostor on the server port is not a Mac to install software
on.

That is the one ending where nothing is written. Something that holds the port
and answers nothing at all is a different case: under per-account data roots
that is what another account's Dessau looks like from here, so the update goes
ahead and the serving version is reported as unknown.

### The panel, again

The administrator panel appears on every single update, for the firewall grant
and nothing else. These builds are ad-hoc signed, so the code identity the
firewall keys its entry to changes with each build, and the entry that covered
the previous build does not cover this one.

Declining it does not fail the run. The new bundle stays in place, the report
says the grant was not made, and it prints the two commands that make it by
hand along with the symptom to expect until they are run: other machines see an
empty response while this Mac works.

### There is no way back

One release is published at a time, so there is no earlier release to fetch and
`dessau update` takes no version. The tag of a previous version survives and
can be rebuilt from source, which is a different job to doing at a terminal.

## Remove it

```sh
dessau uninstall
```

What goes: the application bundle from both fixed locations, the private Python
and MLX runtime, `config.json`, the model list, the logs, the request
statistics, the `dessau` command, and the firewall entry.

What stays: **the models you downloaded, and the cache they arrived through.**
They are the expensive thing to fetch again, so removing them is a decision of
its own. The output states their total size and the flag that removes them.

Two lines are worth reading when they appear:

- **"That was the copy every account on this Mac launches."** The bundle it
  removed was the one in `/Applications`, which is everybody's. No other
  account gets a message about it.
- **What is left.** A path this account may not delete is reported rather than
  elevated for, and the firewall entry is reported with the command that
  removes it when the panel is declined. Everything else is still removed.

`DESSAU_ROOT` and `-root` are never deletion paths. Uninstall acts on the
fixed locations this account's install uses, and the output names the root it
did not remove.

### Remove the models too

```sh
dessau uninstall --purge
```

`--purge` deletes the downloaded models as well. It needs a terminal, because
deleting them is not something to do on somebody's behalf without asking; where
standard input is not a terminal — a script, a pipeline, a CI job — the run
deletes nothing and names the flag that answers for you:

```sh
dessau uninstall --purge --yes
```

### When this Mac uses a shared model cache

With the shared cache from
[Getting started, step 9](getting-started.md#9-sharing-across-user-accounts-optional),
the models live in `/Users/Shared/Dessau` and everything belonging to your
account — its settings, its model list, its runtime, its logs and its
statistics — lives in your own `~/Library/Application Support/Dessau`.

So uninstall removes your own directory and **leaves the shared root alone**.
The output names what remains in it, how much it holds, and how many other
accounts it belongs to, counted rather than named. Removing it removes every
account's models, and it is one deliberate command to run once everybody has
finished with it:

```sh
sudo /bin/rm -rf /Users/Shared/Dessau
```

`--purge` in this mode removes only the files your own account owns, which is
the only removal the shared directory's permissions allow anyway. The output
separates what it deleted from what it left, with the size of each. Another
account's models are that account's to remove.

## Diagnose it

```sh
dessau doctor
```

Doctor runs the expensive checks and labels every line with how much its answer
is worth. Read the label before the severity:

- **Verified** — state Dessau owns, so a severity means what it says: the MLX
  runtime, the data root's writability, the settings file, which build this is,
  and who holds the server port. On a Mac where another account is running the
  server, the port line says so, and says that the build named above it is this
  binary's own rather than the one being served.
- **Observed** — the firewall entry, and it is never a verdict in either
  direction. The system query answers "permitted" for a path it has no entry
  for and for a path that does not exist at all, and an ad-hoc build's identity
  changes with every build, so a "permitted" answer is compatible with a grant
  that covers nothing. The line reports what came back, says what it does not
  establish, and prints the two commands that make the grant again.
- **Cannot be determined from here** — Local Network Privacy, which macOS
  offers no way to read. It has no query interface, is not part of the privacy
  database, and cannot be reset or pre-seeded. The check is printed every time
  rather than dropped, and it carries the command that opens the settings pane
  where a person can look.

Warnings exit zero. Only a check Dessau verified and found wrong exits 1, so
`dessau doctor` is safe to put in a script that branches on the exit code —
and the severity of every finding is in `dessau doctor --json` for a script
that wants more than pass or fail.

For what is true right now rather than what is wrong — serving or not, on which
address, which models are in memory — `dessau status` answers from state that
already exists and is cheap enough to poll.

For what the settings say, without opening the file:

```sh
dessau config show
```

It prints every setting in force, spelled the way `config.json` spells it, so a
figure you want to change by hand can be searched for in the file. It writes
nothing, and the API key and the HuggingFace token are shown as `********` and
never as their values. Add `--json` for a script.

### Pasting the output into a bug report

Doctor's output is written to be pasted. Home directories are abbreviated to
`~`, so no account name travels with it; other accounts on this Mac are counted
rather than named; and the settings file is reported by what it did when it was
read, never by what it says, because it holds the API key and the HuggingFace
token.

The commands it prints are written to be pasted too: a path inside your home
directory is written as `"$HOME/…"`, which runs on the machine it is pasted
back into and carries no account name away.

## Where to go next

- [Reference: the lifecycle verbs](lifecycle-reference.md) — every verb, every
  flag, the exit codes, and the fields `--json` emits.
- [Getting started](getting-started.md) — from a fresh install to answering a
  prompt from another machine.
- [Reference: the server's log](logging.md) — where Dessau writes down what it
  did, and what it never writes.
- [The posture page](posture-reference.md) — the control panel's one page
  saying who can reach this server.

# Check your models for newer versions

A model on HuggingFace changes after you download it: a fixed chat template, a
corrected configuration, a better quantisation. Dessau can check the models you
downloaded for newer versions, so you hear about those fixes rather than finding
out when a client misbehaves. Checks are off until you turn them on.

## Turn checks on

1. Open the control panel and go to **Settings**.
2. Under **Model updates**, tick **Check the models you downloaded for newer
   versions**.
3. Leave **Check every (hours)** blank to check daily, or type a figure from 1
   to 720 (thirty days).
4. **Save settings**.

The first check runs within a few minutes. After that Dessau checks at each
start once the last check is older than the interval, and then at each
interval. The schedule compares the clock with the time of the last check, so a
Mac that slept through a check runs it soon after it wakes.

To turn checks off, untick the box and save. A check already running stops
before it asks about another model. Saving any other setting never turns
checks on.

The same two settings are in `config.json`:

| Key | Value |
| --- | --- |
| `update_check_enabled` | `true` to check; absent or `false` sends nothing. |
| `update_check_interval_hours` | 1 to 720; absent or `0` is daily. A figure outside that range is replaced by the default when Dessau starts, and the control panel says so. |

## What a check sends

For each model you downloaded, a check sends HuggingFace the name of each model
— its repository, such as `mlx-community/Qwen3-8B-4bit` — and asks for that
repository's latest version. It sends no access token, even when one is set in
Settings, unless that repository refuses a request without one: a private or
gated model is then asked again with the token. Nothing else is sent, and
nothing is sent to anyone but HuggingFace. A check never touches the files
you are serving.

## What is checked

- **Only models whose version Dessau recorded.** A download records the
  version it fetched. A model downloaded before Dessau recorded versions, or
  copied into the models folder by hand, is "version unknown" and is not
  checked.
- **Only the files a model server reads.** A newer version that changes only
  a README, a licence or other Markdown is not counted as newer.
- **A newer version that ships its own code is named as such.** If the newer
  version's `config.json` names a `model_file`, Dessau records that it will not
  run that version.

## What the marks mean

Each model's card under **My Models** says what the last check found:

| On the card | What it means | Update |
| --- | --- | --- |
| **newer version** | A newer version changes a file the model server reads. | Offered |
| **newer version not run** | The newer version ships its own code, which Dessau does not run. | Not offered |
| Version unknown | Dessau did not record which version it downloaded, so it cannot tell whether a newer one exists. | Offered: it fetches the current version and records it |
| Nothing | The last check found nothing newer, or checks are off. | Not offered |

Click **Update** to fetch the newer version. The card shows how far it has
come, and **Cancel update** stops it; the model keeps answering requests
meanwhile.

## How a newer version replaces the one you serve

**Update**, like downloading a model you already have, fetches its newer
version beside the one being served, which goes on answering requests the
whole time. Dessau fetches every file at one exact version, checks each
against the hash HuggingFace lists for it, and checks the new version as it
checks any model before it starts one — or, while the runtime Dessau starts
models with is not installed yet, checks the new version's own files, refusing
one that ships its own code, and leaves the rest to the check every model has
before it starts. Only then does it swap the two: it waits for requests
already being answered by the model to finish, moves the new version in, and
keeps the old one aside. While it waits, a new request for that model is answered
at once with `503`, since a model in steady use would otherwise never fall
idle and the update would never land; a client that is told why reads that the
model is being updated and to try again in a moment. The wait lasts at most
two minutes: a request still being answered after that keeps the old version
serving, the update is abandoned, and new requests are answered again. So
update a model that takes long requests when it is quiet. If anything fails
along the way — a file that is not what HuggingFace lists, a download that
stops, a version that ships its own code, or a disk without room for both
versions at once — the new version is removed and the old one keeps serving,
unchanged. The model's card then says why on its version line — a file that
did not download or match, no room on the disk, a version that did not pass
the checks, a model still answering requests when the wait ran out, a
version Dessau does not run, or a newer version that did not load — until an
update succeeds. A file both versions
share is not fetched again.

The old version stays aside until the new one has loaded and answered a
request, since a version can pass every check and still fail to load. The
first request for the model after an update loads the new version: once it
answers, the old version is removed. If that first load fails instead, Dessau
puts the old version back and removes the new one; the request that
triggered the load is refused, the next one is answered by the old version,
and the card says the newer version did not load, with the newer version
still offered. The old version is kept aside across a restart of Dessau too,
but Dessau no longer knows which version it was, so after a restart the
version put back shows as version unknown. Removing the model removes the
old version with it.

What Dessau learned about the old files goes with them: the measured context
window, whether the model calls tools, and a recorded failure to load. What
you set for the model stays: its pin and its served context window.

## When HuggingFace cannot be reached

The check leaves what it recorded last time as it was, writes one line to the
server log, and tries again at the next interval. Serving is not slowed: a
check waits on nothing a request needs.

A check is also cut short when HuggingFace says few requests are left in its
current window; the models it did not reach are checked at the next interval.

# Choose which models are offered for chat

Not every model can hold a conversation. A speech-recognition model, an OCR
model or a base model without a chat template is worth downloading and worth
calling, but it has no business in a chat application's model menu — the first
message sent to one is wasted.

Dessau records what HuggingFace says each model is when you download it, and
publishes those words on the models list. A rule decides which of them count as
able to chat. There are two places to change it: the server, which is what every
client is told, and Dessau Chat, which applies its own. A model the Hub has
said nothing about — one this account found in the shared cache rather than
downloaded — is judged by its own files instead, as [described
below](#a-model-with-no-words).

## On the server

1. Open the control panel and go to **Settings → Which models can chat**.
2. **Pipeline tags that count** is a comma-separated list of HuggingFace
   pipeline tags. As shipped: `text-generation, image-text-to-text`.
3. **Tags a model must carry** is a second comma-separated list, and a model has
   to carry all of them. As shipped: `conversational`.
4. **Save**. The change applies at once — no restart, and nothing is unloaded.

Clear a field to stop testing that half. Clear both and every model is marked as
able to chat, which is the setting to reach for if the rule is hiding models you
want.

The same rule lives in `config.json`, which is hand-editable:

```json
"chat_rule": {
  "pipeline_tags": ["text-generation", "image-text-to-text"],
  "required_tags": ["conversational"]
}
```

Delete the `chat_rule` key to go back to the rule Dessau ships.

Each list holds at most 64 words, of at most 128 bytes each. A save that goes
beyond that is refused and names the field; a `config.json` that does is
repaired on the next start and the panel says which setting was changed, rather
than the server refusing to start.

## In the chat client

Dessau Chat applies its own rule to the same words, so the models it offers are
its user's decision rather than the server's. Open **Settings** (Cmd-,) and
edit the two fields under **Models to offer**. They ship with the server's own
default. The Mac's own model — the client's "On this Mac" — is not a served
model and is exempt from the rule: it is offered whenever the Mac can run it.

## What the rule does and does not do

- **It filters nothing.** Every model stays callable over the API by the name it
  is listed under, whatever the rule says of it. The rule decides one field on
  the models list, `chat`, which a client is free to act on or ignore. See
  [the models list reference](models-list.md).
- **The words are HuggingFace's.** Dessau invents no categories: it republishes
  the repository's pipeline tag and tags as they are written on the Hub, and the
  search tab shows each result's pipeline tag — or "no tag" — before you
  download anything.
- **A model the Hub has tagged, but not as the rule asks, is marked as unable
  to chat** — a speech or OCR model under the shipped rule, or a repository
  whose tags carry no `conversational`. Clear the field that is hiding a model
  you want, or add the words the repository does carry.
- **A model with no words at all is not judged by the rule.** See below.

## A model with no words

A model carries no pipeline tag and no tags when the Hub was never heard for
it: this account did not download it but found it in the
[shared cache](getting-started.md#9-sharing-across-user-accounts-optional),
put there by another account on this Mac; or HuggingFace could not be reached
when the download finished; or a Dessau that predates these words recorded
it. The rule has nothing to test, so it is not consulted. The model's own
files decide instead: a model whose `tokenizer_config.json` carries a
`chat_template`, or that has a `chat_template.jinja` file beside it, counts
as able to chat, and a model with neither does not. The chat template is
what the model's server renders a conversation through, so this is the
question the rule approximates, asked of the files themselves. The models
list carries the answer as `chat_template`, beside `chat`.

Dessau also asks the Hub for the missing words in the background after each
start, one model at a time, and records what it hears; from then on the rule
decides for that model exactly as it does for one this account downloaded.
Nothing waits on this: the server, its models and the chat clients are usable
throughout. A model the Hub cannot be reached for is asked again at the next
start, and one the Hub has no words for is asked once and left alone. The
request is the server's own and appears in no request statistic. Dessau Chat
follows the server's `chat` for a model with no words, since it has no words
to apply its own rule to.

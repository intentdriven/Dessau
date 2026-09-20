# Where a Brave Search API key lives: client or server (SOTA, 2026-09-20)

Asked by the maintainer when proposing Brave search for the product, against
the 2026-09-10 ideate verdict (`2026-09-10-ideate-brave-web-search.md`), which
killed the server-held-key shape and reframed it as an opt-in MCP sidecar.
Researched by an independent agent (evaluator outside the loop); every claim
below carries its evidence tier and a primary source in the list at the end.

## Verdict

**Client-held, per-user key, with Dessau Chat running the search loop.** The
MCP sidecar is a distant second; a server-held operator key in the gateway
stays rejected.

## Ranked

1. **Per-user Brave key in Dessau Chat, loop executed by the client.**
   CONSENSUS on the shape, EVIDENCE on the runtime. Every native desktop peer
   that offers search holds the key on the person's machine and runs the loop
   there: Jan, Msty (the person's own Brave key, desktop-only because Brave
   blocks browser-origin requests), LM Studio and Cline-style clients (Brave
   through MCP with the key in the person's config), Chatbox (bring your own
   key), Ollama's app (the person's own account key). Brave sanctions the
   consumer-holds-own-key shape by practice: its official MCP server and its
   Claude Desktop guide put an individual's key into a third-party desktop
   app. Its terms define "Customer Applications" as apps offered by the
   Customer, which Dessau Chat is not when Alice pastes her own key
   (CONTESTED, a grey area Brave's own guides resolve in the product's
   favour). The pinned mlx-lm 0.31.3 renders `tools` into the chat template,
   returns `tool_calls` with `finish_reason: "tool_calls"` in both modes,
   ignores `tool_choice` and executes nothing; the gateway already passes
   `tools` through untouched (a test holds it). The client has no
   tool-calling code today. What would show it wrong: Brave writing that keys
   inside third-party apps are not Customer Applications; Brave closing
   individual sign-ups; the served models failing to emit tool calls reliably.
2. **Opt-in MCP search sidecar with the operator's key, outside the
   gateway.** CONSENSUS that it is the smallest diff for MCP-capable clients;
   ANECDOTE that anyone on this LAN has one. The operator becomes Brave's
   Customer for every LAN user, must bind each by written agreement, and is
   billed. Does nothing for Dessau Chat unless it grows an MCP client. Kept as
   the recorded fallback, not built first.
3. **Operator key in the gateway, server-side loop.** Rejected, and the
   February 2026 pricing change makes it worse: Brave removed the free tier
   on 12 February 2026; every plan needs a card, gives a small monthly credit,
   then bills per thousand searches. A keyless-default server with one
   operator key is metered spend any LAN client can trigger. Open WebUI is the
   peer that does this and its own docs call the shared key unsuitable for
   multi-user use. LibreChat's per-user key is still server-executed, which
   needs the gateway to read prompt content, which adr-2609061610102325
   forbids.

## Hosted contrast

OpenAI (Responses API only), Anthropic (per-search price, citations returned)
and Google (developer key, mandatory rendering of Search Suggestions for any
grounded result) all run search server-side under the developer's key and
push a display obligation down to the app. The hosted pattern does not remove
the client's display duty; it formalises it.

## Security literature

CONSENSUS: OWASP MASWE-0004 (sensitive data hardcoded in the app package): a
shipped third-party key is extractable and, for per-use-billed services,
leads to financial loss and suspension. The mitigation is the bring-your-own-
key pattern: the person supplies their own credential into platform secure
storage. Dessau Chat already keeps pairing keys in the Keychain, so the store,
its access control and its sync decisions are already made.

## Implications for this product

- **Who holds the key:** each person, in Dessau Chat's Keychain; never in the
  bundle, never in `config.json`, never on the server. Each Mac or iPad user
  needs a Brave account with a card, and the settings screen says so.
- **Who is Brave's Customer:** the person. The maintainer is not a party.
- **What the client displays:** "Powered by Brave" with the logo, in the
  search settings and beside any answer that used results, plus each result's
  title and host.
- **What is never cached:** result payloads. They live in the in-flight
  message array only; the stored conversation keeps the answer and citation
  links, not the snippets (CONTESTED whether on-disk history counts as
  transient; the safe reading is not to persist snippets).
- **What the server must not do:** hold a key, add an outbound host, inspect
  `tool` messages or `tool_calls`, or log query text. It must relay `tool`
  role messages and streamed `tool_calls` deltas untouched; only `tools` has a
  passthrough test today, so two more tests are owed before shipping.
- Because `tool_choice` is ignored, model-initiated search is best-effort per
  model; a deterministic "search first, then ask" mode (fetch, inject as
  context) belongs beside the tool loop.

## Not worth adopting

Brave's "Answers" plan (bypasses the local model); a SearXNG sidecar (a new
server-side outbound host under other engines' terms); the LLM Context
endpoint server-side (same retention, attribution and custody problems).

## Sources

- Brave: terms of service (updated 2026-09-01), privacy notice (2026-08-25),
  pricing and plans, the staff clarification on the monthly credit, the LLM
  Context endpoint, the official MCP server, the Claude Desktop and Open WebUI
  guides — all under `api-dashboard.search.brave.com/documentation/` and
  `brave.com/search/api/`, `github.com/brave/brave-search-mcp-server`.
- Peers: Open WebUI's Brave provider page, LibreChat's web search and env
  pages, Jan's web search page, Msty's real-time data page, Ollama's web
  search blog and docs, AnythingLLM issue 2906, Chatbox's BYOK page, an LM
  Studio MCP tutorial (secondary).
- Hosted: Gemini grounding docs; an OpenAI community thread on BYOK
  (anecdote).
- Security: OWASP MASWE-0004 and MASVS-STORAGE; Apple Keychain services.
- Runtime: mlx-lm v0.31.3 `server.py`; the gateway's `tools` passthrough test
  in `internal/gateway/systemmerge_test.go`.

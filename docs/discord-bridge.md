# Answer Discord messages with a model on this Mac

The Discord bridge lets you message a bot from Discord — on a phone, in a
browser, anywhere Discord runs — and have a model on your Mac answer. Nothing
is installed on the other person's device and no port on your Mac is opened to
the internet.

**What leaves this Mac.** While the bridge is on, messages to the bot and the
model's answers pass through Discord and are kept under Discord's terms. That
is the whole of the trade: Gropius opens one outbound connection to Discord and
relays conversations over it. Everything else about your server is unchanged —
the bind address, the API key, and which machines can reach the OpenAI endpoint
are exactly what they were. Switching the bridge off closes the connection.

The bridge is off until you turn it on.

## Make the bot in Discord

1. Open Discord's developer portal at
   <https://discord.com/developers/applications> and select **New
   Application**. Give it the name people will see.
2. Open the application's **Bot** page.
3. Turn **Public Bot** *off*. While it is on, anybody who finds your
   application can add the bot to their own server, and anyone who can reach
   the bot may talk to it — so where the bot lives is what decides who can use
   your models.
4. Leave the three **Privileged Gateway Intents** off. The bridge does not ask
   for message content, presences or server members, and Discord refuses the
   connection if it is configured to require an intent the bridge does not
   request. A message in a channel that does not mention the bot is never read.
5. Select **Reset Token**, then **Copy**. This is the bot token. Treat it as a
   password: anyone holding it can act as the bot.

## Invite the bot

On the application's **OAuth2** page, build an invite URL with:

- Scopes: `bot` and `applications.commands`. The second is what lets the bot
  register `/model` and `/reset`.
- Bot permissions: **Send Messages**, **Read Message History** and, in a
  thread, **Send Messages in Threads**. Nothing else is needed.

Open the URL and add the bot to a server you control. A direct message to the
bot works without any server at all.

## Paste the token

1. Open the Gropius control panel and go to **Settings**.
2. Find **Discord bridge**, read the sentence beside the switch, and paste the
   token into **Discord bot token**.
3. Turn the switch on and save.

Under the switch the panel says what the bridge is doing: off, connecting,
connected since a moment, or stopped with the reason. A token Discord refuses
stops the bridge and says so there; it never refuses a save, so you can change
any other setting while the bridge is unhappy.

The same two settings are `discord_bridge` and `discord_token` in
`config.json`, and `gropius config show` prints the token redacted, the way it
prints the API key.

## Use it

- **Direct message the bot** and it answers.
- **Mention the bot** in a channel it has been invited to and it answers there.
  A message in that channel that does not mention it is not read at all.
- A reply appears as a placeholder and fills in as the model writes. A long
  answer is cut at a paragraph and continues in a second message.
- Each channel and each direct message is its own conversation, held in memory
  while the bridge runs. Nothing of a message is written to disk.

Two commands work in any channel and in a direct message:

- `/model` shows which model answers there, and lists what this server offers.
  `/model name:<model>` sets it. A channel starts on the first chat model the
  server has.
- `/reset` forgets that channel's conversation and starts again.

## What is recorded

A bridged request is recorded exactly as a request over the API is, with one
extra field: its `source` is `bridge` rather than `http`. The log line carries
the bridge's name, Discord's channel and user identifiers as plain numbers, the
model, the sizes and the timing — never a message, never an answer, never the
token. The
[what is recorded page](statistics-store-reference.md) says all of it field by
field.

## Limits

- The bot answers anyone who can reach it. Control that by where you invite it
  and by leaving **Public Bot** off.
- A conversation is kept to the most recent turns that fit the model's served
  window; older turns drop off rather than the request being refused.
- The bridge holds a limited number of conversations and answers a small number
  of requests at once. A very busy channel has messages dropped rather than
  queued, and the log says so.
- If Gropius cannot serve a request, the bot says so in one short sentence. The
  detailed reason — which model, how much memory, what to change — stays in
  your log, because it describes your Mac.

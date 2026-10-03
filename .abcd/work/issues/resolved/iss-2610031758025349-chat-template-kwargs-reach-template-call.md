---
schema_version: 1
id: "iss-2610031758025349"
slug: "chat-template-kwargs-reach-template-call"
severity: "minor"
category: "security"
source: "agent-finding"
found_during: "audit of mlx-lm 0.32.0's generation thread for iss-2610031444343397"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/gateway/gateway.go"
remedy: "Refuse chat_template_kwargs whose values are not booleans, numbers or strings, or whose keys name one of apply_chat_template's own parameters (chat_template, tokenize, return_*, add_generation_prompt, continue_final_message, tools, documents, tokenizer_kwargs, truncation, max_length, padding)."
resolution: "chat_template_kwargs is refused when it sets one of apply_chat_template's own parameters or carries a non-scalar value."
impact: fix
resolved_by:
  commit: "9086fb03a99800b50bec772e36835c3eb8407e25"
---

chat_template_kwargs is merged into apply_chat_template's own arguments (server.py:550-560): return_dict or return_tensors change what it returns, past _tokenize's guard, so the code after it raises on the generation thread (speculative), and a key such as chat_template runs a client-supplied Jinja template there.

# SOTA — naming a multi-component product family (2026-09-20)

Research for adr-2609200729102059, run by a research agent briefed to break
the session's draft convention rather than confirm it (primary sources: RFC
1178, RFC 6335, RFC 6762, RFC 6763, the IANA service-name registry, the
Apache branding policy, Tailscale and Kubernetes documentation, the
Homebrew and GitLab naming issues, the UNESCO 2019 report on gendered
assistants, the AIES 2024 anthropomorphism paper). Ranked; the
counter-arguments that survived and the name-conflict table follow.

## Findings

1. **The family name Gropius is already an active developer tool.** A
   cross-component issue-management system from the University of Stuttgart
   ships under it: <https://github.com/ccims> (updated 2026-08),
   <https://gropius.dev>, a VS Code extension, tool papers 2020–2024
   (<https://link.springer.com/chapter/10.1007/978-3-030-59155-7_7>,
   <https://dl.acm.org/doi/10.1145/3639478.3639814>). Same audience, same
   word, `.dev` domain and GitHub handle taken. Pre-1.0 is the last cheap
   moment to change it; frozen wire identifiers lock it in at 1.0.
2. **Wire identifiers are protocol, not brand.** RFC 6763 §7: a service name
   "identifies what the service does and what application protocol it
   uses", at most 15 characters, registered per RFC 6335 (name-only
   registration is free). RFC 6763 §6 asks for a `txtvers=` key first in
   the TXT record. Every browser's User-Agent still opening with
   `Mozilla/5.0` is the canonical case of a wire token outliving its brand.
   <https://www.rfc-editor.org/rfc/rfc6763.html>,
   <https://www.rfc-editor.org/rfc/rfc6335.html>.
3. **Product in the service type, machine in the instance name.** RFC 6763
   Appendix D: the instance name "SHOULD be configurable by the user", the
   default "short and descriptive"; Apple's own services default to the
   Computer Name; RFC 6762 §9 resolves duplicates by appending "(2)" and
   persisting it. A product prefix on the instance is legal but redundant.
   Multicast DNS is link-local and does not cross a tailnet; there the
   identifier is the MagicDNS machine name, derived from the hostname.
   <https://datatracker.ietf.org/doc/html/rfc6763#appendix-D>,
   <https://datatracker.ietf.org/doc/html/rfc6762#section-9>,
   <https://tailscale.com/docs/concepts/machine-names>.
4. **Branded house is the peer practice, with one guard.** Tailscale
   (`tailscale`/`tailscaled`, Tailscale SSH, Serve, Funnel), Ollama,
   Kubernetes (`kube-` for first-party parts), Apple's MLX (`mlx`,
   `mlx-lm`, `mlx-swift-lm`), Docker (Compose, Desktop, Hub) all use
   family name plus descriptive component. The documented failure of
   "family name = product name" is Mozilla, which renamed Firefox Monitor,
   Relay and Lockwise to Mozilla Monitor in 2024 when the family outgrew
   the product. So the family name is not the server's display name.
   <https://tailscale.com/docs/reference/tailscale-cli>,
   <https://kubernetes.io/docs/concepts/overview/components/>.
5. **"House of brands works for developer tools" is an endorsed-brand
   argument that costs a marketing organisation.** Apache requires
   "Apache Projectname" as the primary form on every page; HashiCorp moved
   Terraform Cloud to "HCP Terraform" in 2024 to attach the family prefix.
   JetBrains is a genuine house of brands sustained by a large company.
   For one maintainer that is an anti-recommendation.
   <https://www.apache.org/foundation/marks/pmcs>.
6. **Themed hostnames are what RFC 1178 recommends for a handful of
   machines.** "Use theme names"; "Don't choose a name after a project
   unique to that machine", because roles move. Pets-versus-cattle naming
   (`www001`) is for fleets replaced by automation. What survives for this
   product: Computer Name as the instance, roles as aliases.
   <https://www.rfc-editor.org/rfc/rfc1178.html>,
   <http://cloudscaling.com/blog/cloud-computing/the-history-of-pets-vs-cattle/>.
7. **A persona name on the chat client fails every specific check.** UNESCO
   (2019) asked the industry to stop shipping assistants "gendered female
   from the outset by their names"; the anthropomorphism literature (AIES
   2024) documents over-trust driven by human-like cues; trademark law and
   the Apache name guide bar names that falsely suggest a connection with
   persons living or dead, and the Josef & Anni Albers Foundation licenses
   the name (<https://www.albersfoundation.org/rights>); ProjectAnni is an
   active self-hosted media server with a Rust repository named `anni`
   (<https://github.com/ProjectAnni>). Cortana was retired for the
   descriptive Copilot (2023); Bard became Gemini (2024). The one
   counter-example, Rust's Clippy, shows "never a persona on a daemon" is
   not a law either; the real cost driver is the number of distinct names
   a user must hold (finding 8).
   <https://ojs.aaai.org/index.php/AIES/article/download/31613/33780/35677>,
   <https://incubator.apache.org/guides/names.html>.
8. **Themed vocabularies have a documented onboarding cost that scales
   with the number of names.** Homebrew #10798 proposes dropping keg,
   cellar and rack from user-facing docs for "cognitive load from novel
   metaphors"; GitLab #382036 records a term colliding with Kubernetes
   Pods; a 2023 practitioner report describes monthly meetings explaining
   two cute names apart. The useful rule (Tietz-Sokolskaya, 2024):
   hard-to-change names get creative names, easy-to-change ones get
   descriptive names. <https://github.com/Homebrew/brew/issues/10798>,
   <https://gitlab.com/gitlab-org/gitlab/-/issues/382036>,
   <https://ntietz.com/blog/when-to-use-cute-names-or-descriptive-names/>.
9. **Topology in names works when transparent and fails when allusive.**
   `kube-apiserver`, `mlx-swift-lm` and `flamingo-commerce` encode layering
   plainly. Nobody reads "Flamingo" as "the secondary application in the
   plaza of a building by the school's third director". Orphaned names
   (Helm's Tiller, Terraform Cloud, Firefox Monitor) came from a role
   changing under the name. Minting on ship, after a name search, is
   Apple's and Apache's practice.
10. **Themed codenames are a secondary label; the primary identifier is a
    number.** Android dropped dessert names as public labels in 2019
    (not understood everywhere when spoken) but kept them internally;
    Apple moved to year numbers in 2025 and kept the place name as the
    secondary label; Ubuntu pairs adjective-animal with YY.MM.
    <https://9to5google.com/2019/08/22/android-10-dessert-name/>,
    <https://wiki.ubuntu.com/DevelopmentCodeNames>.
11. **A family name can be over-extended.** Nine Microsoft products shared
    "Copilot" with different pricing and data protections, producing
    sustained confusion in Microsoft's own forums after the 2025 hub
    rename. Lesson: stretch the family name only over components inside
    one trust boundary; a client that can talk to any OpenAI-compatible
    server must not imply it is part of the server.
    <https://office365itpros.com/2024/12/20/microsoft-365-copilot-rename/>.

## Counter-arguments that survived

- Themed hostnames are RFC-recommended at this scale; "never themed" was
  wrong. Only "product in the type, machine in the instance" survives.
- Persona names are normal for conversational products; for this product
  (a demonstrator, a woman's name, a real person with an active rights
  holder, an active collision) every specific check comes back negative.
- Descriptive prefixes that encode layering are the ecosystem norm; the
  ban is on allusive topology, not topology.
- A branded house can be over-extended (Copilot, Firefox); the guards are
  "the family name is not the server's display name" and "one trust
  boundary".
- "Do what Apache and HashiCorp do" is endorsed branding backed by policy
  and staff, not a house of brands.

## Name-conflict table

| Name | Software collision | GitHub handle | IANA service name | Persons and marks | Assessment |
|---|---|---|---|---|---|
| Gropius | Active: issue-management system, gropius.dev, github.com/ccims, papers 2020–24 | `gropius` taken | none | Walter Gropius (d. 1969); a Berlin museum carries the name | High: same audience, same word |
| Dessau | none in software; a city-tour app exists | `dessau` dormant since 2019, one repo | none | Geographic name, weak mark; a dissolved Canadian engineering firm dominates search | Low for software |
| Anni | Active: ProjectAnni media server, Rust repo `anni`, updated 2026-09 | `anni` taken | none | Anni Albers (d. 1994); the foundation licenses the name; "anni" is Italian for "years" | Medium-high |
| Mies | AllenInstitute/MIES electrophysiology suite | `mies` is a prominent developer-tools founder | none | Mies van der Rohe (d. 1969), foundation | Medium |
| Flamingo | Crowded: DeepMind Flamingo VLM (2022) reads as a model name inside an LLM product; a Go web framework; an Android Studio release | org unclaimed | none | none found | High in this domain |

Session-side checks on 2026-09-20 for Dessau: Homebrew, PyPI, npm and the
Go package index return nothing; the Avahi service-type database and the
legacy dns-sd.org list carry no `_dessau._tcp`; `_gropius._tcp` is itself
unregistered. Method caveat: the IANA search endpoint returned intermittent
502s during the research session; re-run it once before the registration is
filed. USPTO and EUIPO were not queried (JavaScript-only); do that by hand.

## Verdict

Amend, do not keep as drafted. Keep the branded house with two guards,
freeze and register the wire identifiers, drop the product prefix from the
instance name, put no real person's name on any component, ban allusive
topology only, mint names on ship. Resolve the Gropius collision first.

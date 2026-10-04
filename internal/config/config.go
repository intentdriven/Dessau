// Package config holds Dessau's on-disk layout and user settings.
//
// Everything Dessau creates lives under a single root directory so the whole
// installation — including its private Python interpreter — can be removed by
// deleting one folder.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"net"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
	"unicode/utf8"
)

// Paths is the on-disk layout. All fields are absolute, and every one of them
// is under Root: Dessau serves from one macOS account, and everything it keeps
// — models, executables, settings, state, logs and records — is that
// account's own, in one folder that can be deleted whole.
type Paths struct {
	Root    string // ~/Library/Application Support/Dessau
	Bin     string // uv lives here
	Venv    string // the mlx-lm virtualenv
	Python  string // uv-managed CPython installs (UV_PYTHON_INSTALL_DIR)
	Models  string // downloaded model directories: Models/<org>/<name>
	HFCache string // HF_HUB_CACHE; must exist or mlx_lm.server's /v1/models panics
	// Logs, Config (config.json) and State (registry.json). The settings hold
	// the API key and the HuggingFace token, so config.json is written 0600.
	Logs   string
	Config string
	State  string
	// Stats is where the request statistics store keeps its files: a "stats"
	// directory under the root.
	Stats string
	// SelfTest is where the model self-test keeps its results file: beside
	// Stats, at 0600, for the same reason.
	SelfTest string
}

// userSupportDir is this account's own Dessau directory in Application
// Support — where an installation lives unless DESSAU_ROOT says otherwise.
// It is derived once here so the default root and AccountHome cannot come to
// disagree about where it is.
func userSupportDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, "Library", "Application Support", "Dessau"), nil
}

// DefaultRoot returns where Dessau keeps its data: $DESSAU_ROOT when it is
// set, and this account's own Application Support directory otherwise.
//
// There is no machine-wide root. Dessau serves from one account; every other
// account, on this Mac or elsewhere, reaches it over the network and never
// needs the model files.
func DefaultRoot() (string, error) {
	if env := os.Getenv("DESSAU_ROOT"); env != "" {
		return env, nil
	}
	return InstalledRoot()
}

// InstalledRoot is DefaultRoot without the environment: this account's own
// Application Support directory.
//
// It exists for the one caller that must not honour DESSAU_ROOT — `dessau
// uninstall`, which derives every deletion path from the fixed location this
// account's install actually uses. A directory named through an environment
// variable would otherwise choose what is deleted.
func InstalledRoot() (string, error) { return userSupportDir() }

// AccountHome is this account's own directory, asked for without a data root
// in hand: ~/Library/Application Support/Dessau.
//
// It exists for the swap (internal/lifecycle), which needs somewhere only this
// account can unlink an entry from while a retired application bundle waits to
// be put back, and which has no root to derive it from. It deliberately does
// not honour DESSAU_ROOT: the property wanted here is the one macOS gives
// ~/Library and no environment variable can, so a root named from the
// environment would silently answer with a directory that may not have it.
func AccountHome() (string, error) { return userSupportDir() }

// NewPaths derives the layout from a root directory. Everything is under the
// root, so an installation stays one folder to delete.
func NewPaths(root string) Paths {
	return Paths{
		Root:     root,
		Bin:      filepath.Join(root, "bin"),
		Venv:     filepath.Join(root, "venv"),
		Python:   filepath.Join(root, "python"),
		Models:   filepath.Join(root, "models"),
		HFCache:  filepath.Join(root, "hf", "hub"),
		Logs:     filepath.Join(root, "logs"),
		Config:   filepath.Join(root, "config.json"),
		State:    filepath.Join(root, "registry.json"),
		Stats:    filepath.Join(root, "stats"),
		SelfTest: filepath.Join(root, "selftest"),
	}
}

// UV is the path to the uv binary Dessau manages.
func (p Paths) UV() string { return filepath.Join(p.Bin, "uv") }

// ValidRepoID reports whether s is a well-formed HuggingFace repo id, i.e.
// exactly "<org>/<name>" using only characters HuggingFace itself allows.
//
// This is a security boundary, not a nicety: a repo id flows unmodified into a
// filesystem path (ModelDir) and, once recorded in the registry, into
// os.RemoveAll on delete. A value like "../../../etc" or "a/b/../../.." would let
// a caller escape the models directory. The allow-list (letters, digits, and
// - _ .) matches HuggingFace's own naming rules while forbidding path separators
// beyond the single required "/" and rejecting any "." path segment.
func ValidRepoID(s string) bool {
	org, name, ok := strings.Cut(s, "/")
	if !ok {
		return false
	}
	return validRepoComponent(org) && validRepoComponent(name)
}

// MaxRepoComponent bounds each half of a repo id. HuggingFace itself allows no
// more, and the bound is what turns "at most MaxModels models" into a
// bound on the size of config.json rather than only on its entry count — an
// unbounded key would let a legal number of entries write a file Load then
// refuses to read.
const MaxRepoComponent = 96

// FoldRepoID maps a repo id to the single key that identifies the model it
// names. Two ids naming the same model fold to the same string.
//
// This is the repository's one rule for when two repo ids are the same model,
// and every part that keys anything by a repo id must use it: the registry's
// index, the runtime pool's loaded entries, the models list's join between the
// two, and the downloader's per-model serialization. It lives here, beside
// ValidRepoID, because identity and validity are the same question about the
// same value, and because this package is the one both the registry and the
// runtime already depend on.
//
// It must stay one function rather than one rule copied into several. The
// guarantee the pool and the models list rest on — at most one model per folded
// id — holds only while every site folds identically. Were one site to fold
// more loosely than the registry, two models could share a pool entry and a
// client asking for one would be served the other's weights; were one to fold
// more strictly, a model could be loaded twice.
//
// internal/archtest/repo_id_fold_test.go holds that: in the packages that key
// by a repo id it refuses any case fold that is not on a reasoned allow-list,
// and elsewhere under internal/ it refuses one whose argument is spelled like a
// repo id. The first is what catches a fold hidden behind a local variable; the
// second is a backstop, since those packages legitimately fold file names and
// header names too.
//
// HuggingFace treats repo ids case-insensitively, and ValidRepoID keeps every
// id in the registry to ASCII, so case is the whole of the rule today.
func FoldRepoID(repoID string) string { return strings.ToLower(repoID) }

func validRepoComponent(s string) bool {
	if s == "" || s == "." || s == ".." || len(s) > MaxRepoComponent {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

// VenvPython is the interpreter inside the managed virtualenv.
func (p Paths) VenvPython() string { return filepath.Join(p.Venv, "bin", "python") }

// ModelDir is where a HuggingFace repo id is stored on disk.
// "mlx-community/Qwen3-8B-4bit" -> <Models>/mlx-community/Qwen3-8B-4bit
//
// The repo id must be validated with ValidRepoID first: it becomes a filesystem
// path, and an unvalidated value like "../../.." would escape p.Models and, once
// stored in the registry, could be handed to os.RemoveAll on delete.
func (p Paths) ModelDir(repoID string) string { return ModelDirIn(p.Models, repoID) }

// ModelDirIn is ModelDir for a caller holding only the models directory — the
// registry, which is handed that directory rather than the whole layout.
//
// This is the repository's one rule for where a model's files are, and every
// part that needs a model's directory derives it here from the repo id: the
// download's destination, the delete, the launch, and the registry's scan.
// The `path` registry.json stores beside each model is never what decides it:
// an index written by an earlier layout, or edited by hand, can name a folder
// outside the models directory that still exists, and a model served from it
// would be one this account's own folder does not hold.
func ModelDirIn(models, repoID string) string {
	return filepath.Join(models, filepath.FromSlash(repoID))
}

// EnsureDirs creates every directory in the layout, 0755, and widens nothing.
//
// A layout directory the operator replaced with a symbolic link (models on an
// external disk, say) is followed: the root is this account's own, and so is
// the choice of where its parts live.
func (p Paths) EnsureDirs() error {
	for _, d := range []string{p.Root, p.Bin, p.Venv, p.Python, p.Logs, p.Models, filepath.Dir(p.HFCache), p.HFCache} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", d, err)
		}
	}
	return nil
}

// Config is the user-facing settings file.
type Config struct {
	// Host to bind the gateway to. 0.0.0.0 exposes it to the LAN.
	//
	// It is one of the addresses this server answers on rather than the only
	// one: loopback is acquired alongside whatever this names, so narrowing
	// the bind never costs the operator their own control panel
	// (adr-2609091123526871, iss-7). internal/bind turns this into the set of
	// addresses actually acquired.
	Host string `json:"host"`
	// BindMode decides how the bind is worked out. Empty — the default, and
	// what every configuration written before this field carries — means Host
	// decides. BindModePrivateNetwork means the address is resolved from this
	// Mac's interfaces instead, and Host is left as the operator last set it
	// so that switching back restores their choice.
	//
	// Its own field rather than a sentinel in Host, deliberately: a word like
	// "private" passes host validation as a name, then fails to listen, and
	// the app exits with no panel and no recovery but editing the file.
	BindMode string `json:"bind_mode"`
	Port     int    `json:"port"`
	// TLSPort is the second port, where paired clients speak to this server
	// over TLS and prove themselves with a key of their own
	// (adr-2609182357322050). Zero — the default, and what every configuration
	// written before this field carries — means the port beside Port. A
	// negative value means no TLS listener, which is what a port this Mac
	// cannot serve is narrowed to at load.
	TLSPort int `json:"tls_port,omitempty"`

	// APIKey, when non-empty, requires "Authorization: Bearer <key>" on /v1
	// requests. Empty (the default) means the LAN endpoint is open.
	APIKey string `json:"api_key"`

	// UpstreamHeaderTimeoutSec bounds how long the gateway waits for a model
	// server to return response headers, i.e. to finish prefill.
	//
	// Zero means automatic: the bound is derived from the prompt's size, which
	// is what actually decides prefill time. A fixed bound cannot be right for
	// both a 4K prompt and a 256K one — measured prefill on this hardware falls
	// from about 1,300 tokens per second at 8K to under 200 at the largest
	// verified sizes, so a bound sized from small prompts is wrong by two to
	// four times exactly where it matters.
	//
	// A positive value overrides the derivation with a fixed number of seconds.
	// It exists because the derivation encodes a measurement, and a measurement
	// can be wrong for hardware or a model nobody tested: an operator who finds
	// it so must be able to say so without waiting for a release.
	UpstreamHeaderTimeoutSec int `json:"upstream_header_timeout_sec"`

	// Advertise the service over Bonjour/mDNS so other machines can find it.
	Advertise bool `json:"advertise"`

	// IdleTimeoutSec unloads a model server after this long with no requests.
	// Zero keeps models resident forever.
	IdleTimeoutSec int `json:"idle_timeout_sec"`

	// DecodeConcurrency maps to mlx_lm's --decode-concurrency: how many
	// requests get batched together during token generation. The default is
	// one, because each sequence holds its own attention cache and the
	// default served window is derived from what the budget has left per
	// sequence: at one a 128 GB Mac serves its long-context models at
	// windows of over 100k tokens, at four at a quarter of that. A saved
	// figure is the operator's and a change of default never rewrites it.
	DecodeConcurrency int `json:"decode_concurrency"`

	// HFToken authenticates against gated HuggingFace repos.
	HFToken string `json:"hf_token"`

	// DiscordBridge switches the Discord bridge on. Off until the operator
	// turns it on, which is adr-2609181004167097 condition 1: nothing of a
	// conversation leaves this Mac until they have pasted a token and thrown
	// this switch. Off closes the connection.
	DiscordBridge bool `json:"discord_bridge"`

	// DiscordToken is the bot token the bridge identifies with — a bearer
	// credential, and the third secret this file holds. It is redacted
	// wherever the API key is, round-tripped through the panel's placeholder,
	// and never compared against a posted value (adr-2609181004167097
	// condition 3, the three obligations itd-2609081259493890 established for
	// the key).
	//
	// NOTHING HERE REFUSES A SAVE. A token that is absent, malformed or
	// rejected by Discord is the bridge's problem and the bridge's alone to
	// report: Validate says nothing about it, so an operator saving an
	// unrelated setting is never turned away over a credential they did not
	// touch. What the bridge will not accept it says on the panel, as its own
	// state.
	DiscordToken string `json:"discord_token"`

	// Preload lists repo ids to load into memory at startup, so the first request
	// after a restart is not a multi-minute cold start. Loaded sequentially and
	// best-effort — an invalid or too-large entry is logged and skipped, never
	// blocking startup.
	Preload []string `json:"preload,omitempty"`

	// MaxResidentBytes caps the total charged size of the models that may be
	// in memory at once. Zero — the default, and what a fresh install stores —
	// means a share of this Mac's physical memory, resolved where that size is
	// known; it is not a second representation of the same figure.
	//
	// Bytes, because that is what the pool's own messages already speak in and
	// what a load is measured against; the panel types gigabytes and converts.
	// Validate keeps to the one rule that holds on every Mac — the figure must
	// not be negative — since a ceiling checked here would make a config.json
	// written on a large Mac invalid on a smaller one, and an invalid config
	// takes the whole install down to loopback with the shipping defaults.
	// The ceiling belongs where the machine's size is known, at a settings
	// save.
	MaxResidentBytes int64 `json:"max_resident_bytes,omitempty"`

	// EvictionGrace turns on the bounded wait a request pays instead of
	// evicting a model that has only just finished work. It is off unless the
	// operator turns it on, and while it is off a request for a model that
	// does not fit takes an eviction victim at once, exactly as it always has.
	EvictionGrace bool `json:"eviction_grace,omitempty"`

	// EvictionGraceSec is how long a model is protected after it finishes a
	// request, and EvictionMaxWaitSec is the longest a request will wait for
	// room before it is refused. Both are whole seconds and both apply only
	// while EvictionGrace is on.
	//
	// Zero means the default, the way it does for MaxResidentBytes: a field
	// cleared in Settings, a file written by a build that had no such field,
	// and a fresh install all mean the same thing, and none of them may stand
	// between the operator and saving an unrelated setting. Read them through
	// GraceSeconds and MaxWaitSeconds, which resolve that.
	//
	// The grace may not exceed a non-zero IdleTimeoutSec: the idle reaper
	// would otherwise unload the very model a wait is protecting, which would
	// make the promise false.
	EvictionGraceSec   int `json:"eviction_grace_sec,omitempty"`
	EvictionMaxWaitSec int `json:"eviction_max_wait_sec,omitempty"`

	// Sampling holds the machine-wide sampling defaults every model server is
	// launched with, so a request that omits a parameter is served with them.
	Sampling Sampling `json:"sampling,omitzero"`

	// Statistics turns on content-free recording of the requests this Mac
	// serves: which model, how it ended, how many tokens and how long it took.
	// It is off until the operator turns it on, and while it is off nothing
	// worked out from a request is recorded or shown
	// (adr-2609061503319212). Nothing recorded leaves the Mac, and no prompt,
	// completion, key or client address is ever part of it.
	Statistics bool `json:"statistics,omitempty"`

	// StatsMonths and StatsMaxBytes bound how much of that record is kept on
	// disk: a horizon in whole months, and a hard ceiling in bytes. The
	// ceiling always wins — the months figure prunes within it — so the store
	// is bounded however busy the Mac is, and how far back it actually reaches
	// is shown in Settings rather than promised (adr-2609061610107154).
	StatsMonths   int   `json:"stats_months,omitempty"`
	StatsMaxBytes int64 `json:"stats_max_bytes,omitempty"`

	// SelfTest lets Dessau measure its own models while nobody is using it:
	// when the Mac has been idle for a while it loads each downloaded model in
	// turn, runs the same short set of tests against it, records the figures
	// in a file under this account's data directory, and unloads what it
	// loaded (itd-2609100457007827). Off until the operator turns it on, and
	// while it is off nothing is loaded and nothing is written. The figures
	// hold no prompt and no answer, and nothing recorded leaves the Mac
	// (adr-2609061503319212).
	SelfTest bool `json:"self_test,omitempty"`

	// ContextProbe lets Dessau measure each model's servable context window
	// while nobody is using the Mac: prompts of growing size through its own
	// OpenAI endpoint, bisected to the largest the server accepts, recorded
	// on the model's registry entry and published beside the declared and
	// the served window (itd-2609091301112705). Off by default; a per-model
	// "Measure now" runs one probe whatever this says. A measurement changes
	// no charge and refuses no request until the operator adopts it as the
	// served window.
	ContextProbe bool `json:"context_probe,omitempty"`

	// IdleThresholdSec is how long the last request must be in the past
	// before the Mac counts as idle for the self-test and the context probe
	// — one idea of idle for both. Zero means the default; read it through
	// EffectiveIdleThresholdSec.
	IdleThresholdSec int `json:"idle_threshold_sec,omitempty"`

	// APIUnloadOff turns off POST /v1/dessau/unload, the model API's request
	// for a program to unload a model it has finished with
	// (adr-2610031153127219). Absent means on: the route is open only to the
	// callers this server already trusts, and only for a model nobody is
	// relying on. The control panel's own Unload is not affected.
	APIUnloadOff bool `json:"api_unload_off,omitempty"`
	// UpdateCheck lets Dessau ask HuggingFace, at start when the last check
	// is older than the interval and at each interval, whether each model it
	// downloaded has a newer version, and mark the ones that have
	// (itd-2610030857275099). Off until the operator turns it on, and while it
	// is off no check leaves the Mac (adr-2610030857208746). A check sends
	// the repository's name and no token unless the repository refuses an
	// anonymous request.
	UpdateCheck bool `json:"update_check_enabled,omitempty"`
	// UpdateCheckIntervalHours is how often the check runs, from one hour to
	// thirty days. Zero means daily; read it through
	// EffectiveUpdateCheckInterval.
	UpdateCheckIntervalHours int `json:"update_check_interval_hours,omitempty"`

	// LogLevel decides how much Dessau writes about itself, in its own log and
	// on standard error. "sparse" — the default, and what every configuration
	// written before this field carries — is one line per event that mattered:
	// a refusal, a model loading or leaving, a launch that failed, the server
	// starting and stopping, a settings save. "detailed" adds the figures those
	// lines omit: how many requests were already in flight, the memory budget
	// in bytes, how long a request waited, the wrapped launch error, and the
	// drain behind an eviction.
	//
	// It is Dessau's own level and reaches nothing else. In particular it
	// never reaches the model servers, which are launched at INFO whatever this
	// says (adr-2609061503319212, and the guard in
	// internal/archtest/statistics_switch_test.go): at DEBUG mlx_lm writes
	// prompts and completions to its log, and no setting in this file may ask
	// for that.
	//
	// Empty means sparse, the way an empty BindMode means Host decides. Read it
	// through EffectiveLogLevel, so "unset" has one meaning and not one per
	// caller.
	LogLevel string `json:"log_level,omitempty"`

	// ChatRule decides which models are published on the models list as able
	// to hold a conversation, from the Hub's own words for what a model is.
	// Absent — the default, and what a fresh install stores — means the rule
	// Dessau ships (DefaultChatRule). It is not a second representation of
	// that rule: a rule whose two lists are present and empty tests nothing,
	// which is how an operator says "offer every model for chat".
	//
	// Machine-wide, and deliberately not one of the per-model settings below:
	// it is one rule read against every model's own words, not a thing the
	// operator says model by model. It filters nothing either way — every
	// model stays callable by name over the API whatever the rule says of it.
	ChatRule ChatRule `json:"chat_rule,omitzero"`

	// Models holds every setting that belongs to one model rather than to the
	// machine, keyed by the registry's canonical repo id. A model with no
	// entry runs on the machine-wide settings above, which is what every model
	// does until the operator says otherwise.
	//
	// One map, and exactly one. Dessau carried three of these — a sampling
	// override map, a per-model settings map and a pinned list — each with its
	// own ceiling, its own sanitiser, its own guard in the settings handler
	// and its own canonicalisation, held to the same rules by prose in three
	// files (iss-2609062213413447). A new per-model setting is a field on
	// ModelSettings, and internal/archtest holds the count at one.
	Models map[string]ModelSettings `json:"models,omitempty"`
	// Clients is the paired set, keyed by each client's key fingerprint
	// (adr-2609182357322050). It is a setting in this file and reached from the
	// panel's own Clients pane, but it is NOT part of the settings form's round
	// trip: a settings save can neither add, alter nor remove a client.
	// Pairing and revoking are routes of their own, which is what keeps the
	// loopback-only settings endpoint from being a second, credential-free way
	// to write who may connect.
	Clients map[string]Client `json:"clients,omitempty"`
}

// ModelSettings are the settings of a single model: everything Dessau does
// differently for one model rather than for the machine.
//
// Every field is off or zero by default, so a model gains a behavior only when
// the operator switches it on for that model in Settings.
//
// A file written by a newer build still loads: a setting this build does not
// know is ignored and the ones it does know are unaffected. It does not
// survive a save, though — this build re-marshals what it holds, so running an
// older build and saving settings drops a newer build's per-model fields for
// good. That is the same bargain every field in this file has always made, and
// it is why a rollback is a decision rather than a shrug.
type ModelSettings struct {
	// MergeSystemMessages folds every system-role message of a chat completion
	// request for this model into one leading system message before the request
	// reaches the model server, which is the shape a chat template that refuses
	// a system message anywhere but the front will accept.
	//
	// It is the one case in which Dessau reads the content of a request's
	// messages, it reads them for no other purpose, and it keeps nothing it
	// reads (adr-2609061610102325). Off unless the operator switches it on for
	// this model.
	MergeSystemMessages bool `json:"merge_system_messages,omitempty"`

	// Pinned keeps this model in memory: a pinned model is never chosen as an
	// eviction victim and is never unloaded by the idle timeout, so a request
	// that would need its memory is refused instead.
	//
	// Separate from Preload, and the two do different things. Preload loads a
	// model at startup and leaves it as evictable as any other; pinning
	// protects a model but loads nothing, so a pinned model is protected from
	// the moment something loads it. A model in both is loaded at startup and
	// protected from then on.
	Pinned bool `json:"pinned,omitempty"`

	// Sampling overrides the machine-wide sampling defaults for this model. A
	// parameter it does not name keeps the machine-wide value, so an override
	// that says only "temperature 0.2" still gets the machine's token budget.
	Sampling Sampling `json:"sampling,omitzero"`

	// ServedContext is the context window Dessau serves this model at, in
	// tokens. Zero means the default, which is not stored: the largest window
	// that fits the memory budget at the decode concurrency in force, capped
	// at the window the model's own configuration declares, worked out by
	// App.ServedWindow in internal/app from figures only the app holds
	// together.
	//
	// It is one figure with two effects, and that is the point of it: the
	// memory budget charges the attention cache this window costs, and the
	// gateway refuses a request estimated to be larger than it. Lowering it is
	// how a model whose default window is too short for the operator's use is
	// traded against the memory it costs. Read through
	// Config.ServedContextSetting, never off this field, so that the folding
	// and the cap are applied in one place.
	ServedContext int64 `json:"served_context,omitempty"`

	// NoTranscript takes this model out of the recording
	// (itd-2609091715089488): while it is set, nothing this model is asked
	// and nothing it answers is written to the transcript store, whether or
	// not the machine-wide switch is on. Off unless the operator sets it for
	// this model. Read through Config.NoTranscript, never off this field, so
	// that the folding is applied in one place.
	NoTranscript bool `json:"no_transcript,omitempty"`
}

// MaxContextLength bounds every context window Dessau will believe, declared
// or served. A model directory's config.json is whatever the repository it
// was downloaded from says, and config.json itself is edited by hand; the figure is
// served to the LAN and decides how much memory a model is charged, so a
// hostile or corrupt one must not be able to hand a client an absurd number to
// size buffers from or fill this Mac's memory with. 8,388,608 tokens is far
// above any window in use and far below anything that could be mistaken for
// one. The registry bounds a declared window by this same constant.
const MaxContextLength = 1 << 23

// IsZero reports whether a model's settings say nothing at all. An entry like
// that is dropped rather than stored — by sanitizeModels on the way in from
// the file, and by the settings path on the way in from a save — so an empty
// object neither holds a slot against the ceiling nor reaches config.json.
//
// Declared rather than inherited: Sampling has an IsZero of its own, and an
// embedded or promoted one would report a pinned model with no sampling
// override as having no settings.
func (m ModelSettings) IsZero() bool {
	return !m.MergeSystemMessages && !m.Pinned && m.ServedContext == 0 && m.Sampling.IsZero() && !m.NoTranscript
}

// ServedContextSetting is the operator's served window for the named model,
// or 0 when they have set none. It is the reader of the SETTING and not of
// the served window: the window a model is served at is App.ServedWindow in
// internal/app, which derives a default from the memory budget when this
// answers 0, and every surface — the charge, the gateway's refusal, the
// models list, the panel — reads that. A second reading of the setting
// anywhere else is how those come to mean different windows by one number,
// which is why an architecture test holds the callers of this to
// internal/app.
//
// A setting above the declared window is not honoured as typed: the operator
// can ask for less than the model was built for and cannot ask for more, so
// the answer is the declared window — a figure of the operator's, capped,
// rather than no setting at all.
func (c Config) ServedContextSetting(repoID string, declared int64) int64 {
	set := c.Models[repoID].ServedContext
	if set == 0 {
		// Folded, because a request resolves to the registry's spelling and
		// the settings file is written by hand as often as by the panel.
		//
		// Every variant is read and the largest kept, rather than the first
		// the map hands over. Two spellings of one id are refused on the
		// settings path and dropped on the file path, so a map holding both
		// reached here some other way — assembled in Go, or written by a build
		// with different rules — and taking whichever came first would answer
		// differently on different runs of the same binary. The largest is the
		// one choice that is both deterministic and no smaller than what the
		// operator asked for anywhere.
		folded := FoldRepoID(repoID)
		for id, ms := range c.Models {
			if FoldRepoID(id) == folded && ms.ServedContext > set {
				set = ms.ServedContext
			}
		}
	}
	if set <= 0 {
		return 0
	}
	if declared > 0 && set > declared {
		return declared
	}
	return set
}

// NoTranscript reports whether this model is excepted from the recording
// (itd-2609091715089488). Every path that decides whether to write goes
// through this reader, never through a raw index into Models.
//
// Folded, as ServedContextSetting is: a request resolves to the registry's
// spelling and the settings file is written by hand as often as by the
// panel. Every variant is read and ANY that carries the exception wins.
// Two spellings of one id are refused on the settings path and dropped on
// the file path, so a map holding both reached here some other way; the
// reader answers the same on every run of the same binary either way, and
// it fails closed — the operator asked for nothing to be written.
func (c Config) NoTranscript(repoID string) bool {
	if c.Models[repoID].NoTranscript {
		return true
	}
	folded := FoldRepoID(repoID)
	for id, ms := range c.Models {
		if ms.NoTranscript && FoldRepoID(id) == folded {
			return true
		}
	}
	return false
}

// MergeSystemMessages reports whether the model's system messages are merged
// into one, read the way every per-model setting is: folded, so a setting
// stored under another spelling of the model's id than the registry's still
// applies (iss-2609201015464436). Any spelling that carries it wins, as for
// NoTranscript; two spellings of one id cannot both reach here from the
// file or a save, which drop and refuse them, so the choice is moot.
func (c Config) MergeSystemMessages(repoID string) bool {
	if c.Models[repoID].MergeSystemMessages {
		return true
	}
	folded := FoldRepoID(repoID)
	for id, ms := range c.Models {
		if ms.MergeSystemMessages && FoldRepoID(id) == folded {
			return true
		}
	}
	return false
}

// Clone returns a copy that shares no pointer with the original — the sampling
// override's fields are pointers, because a blank field and a zero are
// different answers.
func (m ModelSettings) Clone() ModelSettings {
	m.Sampling = m.Sampling.Clone()
	return m
}

// ValidateModelKeys reports whether every key of a per-model settings map
// names a model, i.e. is a well-formed "<org>/<name>" repo id.
//
// A key is matched against the id a request resolves to, so a key of any other
// shape names nothing and would sit in the settings file looking effective
// while applying to no request ever made. Refusing it at the point of saving
// is the only moment the operator is there to see it.
func ValidateModelKeys(m map[string]ModelSettings) error {
	// Sorted, so a file with several unusable keys names the same one every
	// time it is refused rather than whichever the map iteration reached first.
	for _, id := range modelKeys(m) {
		if !ValidRepoID(id) {
			return fmt.Errorf("settings for model %q: not a model id of the form <org>/<name>", id)
		}
	}
	if len(m) > MaxModels {
		return fmt.Errorf("per-model settings name %d models, more than the %d this holds", len(m), MaxModels)
	}
	return nil
}

// MaxModels bounds the per-model settings map: everything saved is written to
// config.json, which Load refuses above MaxConfigBytes, and a config.json that
// cannot be read sends the next start into its fail-closed loopback-only
// branch, taking the LAN endpoint with it. A bounded map keeps this field from
// being the lever for that, whether it is filled from the control plane or by
// hand in the file. Nobody has hundreds of models on one Mac.
//
// The memory budget bounds how many models can usefully be pinned, but
// Validate is machine-independent, so the count is what is bounded here.
const MaxModels = 256

// modelKeys returns a per-model settings map's keys in a stable order, so that
// a map with more than one problem in it names the same one every time rather
// than whichever the map iteration reached first.
func modelKeys(m map[string]ModelSettings) []string {
	return slices.Sorted(maps.Keys(m))
}

// PinnedIDs names the models pinned in memory, in a stable order.
//
// The pool, the fit check and the panel all take a list; the settings hold a
// map, because pinning is a setting of one model like any other. This is the
// one place the two shapes meet.
func (c Config) PinnedIDs() []string {
	var out []string
	for _, id := range modelKeys(c.Models) {
		if c.Models[id].Pinned {
			out = append(out, id)
		}
	}
	return out
}

// validateModels checks the per-model settings the way the machine-wide ones
// are checked: this is the settings path, where a human is waiting for an
// answer, so an entry that names no model is refused rather than dropped.
func (c Config) validateModels() error {
	if err := ValidateModelKeys(c.Models); err != nil {
		return err
	}
	seen := map[string]string{}
	for _, id := range modelKeys(c.Models) {
		// Two spellings of one repo id are two entries in the map but one
		// model, so the effective settings would depend on which the lookup
		// reached first. sanitizeModels drops the duplicate on the file path;
		// here, where a human is waiting for an answer, say so instead.
		folded := FoldRepoID(id)
		if first, ok := seen[folded]; ok {
			return fmt.Errorf("settings for %q and %q name the same model", first, id)
		}
		seen[folded] = id
		if err := c.Models[id].Sampling.Validate(); err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		if sc := c.Models[id].ServedContext; sc < 0 || sc > MaxContextLength {
			return fmt.Errorf(
				"%s: served_context must be between 0 (the model's own window) and %d, got %d",
				id, int64(MaxContextLength), sc)
		}
	}
	return nil
}

// sanitizeModels drops every per-model entry this build cannot use — and every
// value inside one the model server would refuse — returning what it dropped,
// so a settings file written by hand, restored from a backup, or produced by
// another build still loads.
//
// Refusing the file instead would be worse than useless. The panel serves the
// stored settings into its form and the form posts them back, so one unusable
// key would return on the next save and be refused there — wedging every
// settings change there is, the API key included, until someone edited the
// file by hand.
func (c *Config) sanitizeModels() []string {
	if len(c.Models) == 0 {
		return nil
	}
	var dropped []string
	kept := make(map[string]ModelSettings, len(c.Models))
	seen := map[string]string{} // folded id -> the spelling kept
	for _, id := range modelKeys(c.Models) {
		if !ValidRepoID(id) {
			dropped = append(dropped, "models["+id+"]")
			continue
		}
		// Two spellings of one repo id would make the effective settings
		// depend on map iteration order. Keep the first in sorted order so the
		// outcome is the same on every start.
		folded := FoldRepoID(id)
		if first, ok := seen[folded]; ok {
			dropped = append(dropped, "models["+id+"] (duplicate of "+first+")")
			continue
		}
		if len(kept) >= MaxModels {
			dropped = append(dropped, "models["+id+"] (beyond the "+
				strconv.Itoa(MaxModels)+"-model ceiling)")
			continue
		}
		ms := c.Models[id].Clone()
		sampling, names := ms.Sampling.Sanitized()
		ms.Sampling = sampling
		for _, n := range names {
			dropped = append(dropped, "models["+id+"].sampling."+n)
		}
		// A window this build cannot use is dropped on its own, leaving the
		// model's other settings in force: the alternative is a file whose one
		// bad figure takes a pin and a sampling override down with it.
		if ms.ServedContext < 0 || ms.ServedContext > MaxContextLength {
			ms.ServedContext = 0
			dropped = append(dropped, "models["+id+"].served_context")
		}
		// An entry that says nothing is not kept, and is not reported either:
		// nothing was ignored, because nothing was asked for. Keeping it would
		// let empty objects fill the map to its ceiling and stand between the
		// operator and a model they do want settings for.
		if ms.IsZero() {
			continue
		}
		seen[folded] = id
		kept[id] = ms
	}
	if len(kept) == 0 {
		kept = nil
	}
	c.Models = kept
	return dropped
}

// MaxPreload bounds the preload list, for the reason MaxModels bounds the
// per-model settings beside it and to the same figure: a config.json grown
// without limit is one the next start cannot read, and a start that cannot
// read it locks the server down to loopback. The list names models to load
// into memory at startup, so it cannot usefully be longer than the models this
// Mac can hold — a bound the memory budget puts in the single digits — while
// Validate has to be machine-independent, so what is bounded here is the
// count, at the figure the per-model settings already use.
const MaxPreload = MaxModels

// MaxAPIKeyBytes bounds the API key, for the same reason and against the same
// hazard.
//
// GenerateAPIKey produces 43 characters (32 random bytes, base64 without
// padding), which is what a fresh install carries and what almost every
// install keeps. The ceiling is an order of magnitude above that, so a
// passphrase a person chose or a password manager produced fits with room to
// spare, and the field stops being a lever for growing config.json towards
// MaxConfigBytes from the settings endpoint.
const MaxAPIKeyBytes = 512

// sanitizeBudget drops a memory budget this build cannot use and names what it
// dropped, so a hand-edited file still loads.
//
// Falling back to the default is the whole point: refusing the file would send
// main into its fail-closed loopback-only branch, so the API key and the bind
// address the operator set would go unused over one figure that has a perfectly
// good default sitting behind it.
func (c *Config) sanitizeBudget() []string {
	if c.MaxResidentBytes >= 0 {
		return nil
	}
	dropped := []string{"max_resident_bytes[" + strconv.FormatInt(c.MaxResidentBytes, 10) + "]"}
	c.MaxResidentBytes = 0
	return dropped
}

// sanitizePreload cuts a preload list this build cannot use down to the
// ceiling and names what it cut, so a hand-edited file, a backup or another
// build's settings still load.
//
// Cut rather than refused, for the reason sanitizeModels gives: the panel
// serves the stored settings into its form and the form posts them back, so a
// list that is refused rather than trimmed would come back on the next save
// and be refused there — wedging every settings change there is, the API key
// included, until someone edited the file by hand.
//
// The entries kept are the ones at the front. They are all equally
// well-formed — only their position is the problem — so one line naming the
// field and the count says everything an operator can act on, where a line per
// entry would say the same thing two hundred times.
func (c *Config) sanitizePreload() []string {
	if len(c.Preload) <= MaxPreload {
		return nil
	}
	cut := len(c.Preload) - MaxPreload
	c.Preload = c.Preload[:MaxPreload]
	return []string{"preload (" + strconv.Itoa(cut) + " entries beyond the " +
		strconv.Itoa(MaxPreload) + "-model ceiling)"}
}

// sanitizeAPIKey trims an API key this build cannot use and says that it did,
// so a hand-edited file, a backup or another build's settings still load.
//
// Trimmed rather than cleared: clearing it would turn one over-long value in a
// file into a LAN-exposed server that anyone on the network can use, which is
// the one outcome this configuration is never allowed to arrive at by
// accident. Trimmed rather than refused, for the reason the sanitizers above
// give — a refused file locks the next start down to loopback, and the key
// would come back on the next save and be refused there.
//
// What is left is still a key, and a client using the old one is refused: the
// file was already carrying a value no save would have written, and the panel
// shows the operator exactly what is in force now. The value is never named in
// what is reported — it is the secret this field exists to hold.
func (c *Config) sanitizeAPIKey() []string {
	if len(c.APIKey) <= MaxAPIKeyBytes {
		return nil
	}
	key := c.APIKey[:MaxAPIKeyBytes]
	// A cut through the middle of a multi-byte character would leave a string
	// that is not valid UTF-8, which Save would re-encode as something else
	// again. Step back to the last whole character; at most three steps for a
	// value that is UTF-8 at all.
	whole := key
	for len(whole) > 0 && !utf8.ValidString(whole) {
		whole = whole[:len(whole)-1]
	}
	// A value with no whole character anywhere in it leaves nothing to step
	// back to, and the raw cut is kept rather than the empty string this would
	// otherwise produce. Nothing reaches here with one today — Load's decoder
	// coerces invalid UTF-8 to U+FFFD — and the guard does not depend on that
	// staying true: an empty key is an open server on a LAN-exposed install,
	// which is the one repair this function must never make.
	if whole != "" {
		key = whole
	}
	c.APIKey = key
	return []string{"api_key (trimmed to the " + strconv.Itoa(MaxAPIKeyBytes) + "-byte ceiling)"}
}

// Clone returns a copy that shares no slice, map or pointer with the original.
//
// The settings endpoint decodes a posted body into a copy of the live config
// so that fields the form does not own keep their values. A shallow copy is
// not enough for that: encoding/json writes through an existing non-nil
// pointer into its target, and reuses an existing slice's and map's storage,
// so a posted value would reach the running configuration before Validate had
// a chance to refuse it — and would stay there once it had.
func (c Config) Clone() Config {
	out := c
	if c.Preload != nil {
		out.Preload = append([]string(nil), c.Preload...)
	}
	out.Sampling = c.Sampling.Clone()
	if c.Models != nil {
		out.Models = make(map[string]ModelSettings, len(c.Models))
		for k, v := range c.Models {
			out.Models[k] = v.Clone()
		}
	}
	// Both halves of the rule: the settings write path decodes a posted body
	// into a clone, and a shared backing array would land a caller's words in
	// the live rule before Validate had looked at them.
	//
	// ChatRule.Clone copies with make and copy rather than appending onto a nil
	// slice, which for an empty half would yield nil — turning a rule an
	// operator cleared back into a rule they never set, and so reinstating the
	// shipped default at the next save of any unrelated setting.
	out.ChatRule = c.ChatRule.Clone()
	// The paired set is the list of who may connect. A shared map would land a
	// posted body in it before Validate had looked at it, and leave it there
	// when the save was refused.
	if c.Clients != nil {
		out.Clients = make(map[string]Client, len(c.Clients))
		for k, v := range c.Clients {
			out.Clients[k] = v
		}
	}
	return out
}

// Retention bounds for the statistics store.
//
// The defaults are the ADR's starting values, with its arithmetic corrected by
// measurement: internal/stats' BenchmarkLatestAtTheCap fills the cap and
// reports about 380 bytes a record — a JSON Lines record carries its field
// names on every line — so 200 MB is roughly three months of ten thousand
// requests a day, not the four the ADR reasoned to from 150 bytes. The size
// cap therefore bites at about half the six-month horizon on a Mac that busy,
// which is what makes it the hard bound rather than a formality. The floor
// on the ceiling is two rotated files, below which the store would drop a file
// it had only just opened; the ceiling on the ceiling and the horizon are
// there so a mistyped figure is refused rather than filling a disk or being
// read as "forever". Ten gigabytes is decades of records at the rate above,
// and small enough that a mistyped figure cannot quietly become the whole
// disk — nothing here asks the filesystem how much room is left.
const (
	DefaultStatsMonths   = 6
	DefaultStatsMaxBytes = 200 << 20
	MinStatsMaxBytes     = 10 << 20
	MaxStatsMaxBytes     = 10 << 30
	MaxStatsMonths       = 120
)

// Bounds for the two eviction-grace intervals.
//
// The defaults are the record's: 120 seconds covers the pause an agent takes
// between turns, which is the eviction this feature exists to prevent, and 300
// seconds is longer than most clients will wait but short enough that a
// request that will never be served fails rather than hangs. The ceiling is an
// hour: past that the wait is longer than any interactive client's own
// timeout, so a figure above it is a typing mistake rather than a policy. Zero
// is not a figure but the absence of one, and resolves to the default: see
// Config.GraceSeconds.
const (
	DefaultEvictionGraceSec   = 120
	DefaultEvictionMaxWaitSec = 300
	MaxEvictionWaitSec        = 3600
)

// validateGrace holds the two intervals to their ranges and to the idle
// timeout.
//
// The idle rule applies only while the switch is on: with grace off the two
// figures decide nothing, and refusing a save over an inert pair would put a
// setting the operator is not using between them and the API key they came to
// set.
func (c Config) validateGrace() error {
	if c.EvictionGraceSec < 0 || c.EvictionGraceSec > MaxEvictionWaitSec {
		return fmt.Errorf("the eviction grace must be between 0 and %d seconds, got %d",
			MaxEvictionWaitSec, c.EvictionGraceSec)
	}
	if c.EvictionMaxWaitSec < 0 || c.EvictionMaxWaitSec > MaxEvictionWaitSec {
		return fmt.Errorf("the maximum wait must be between 0 and %d seconds, got %d",
			MaxEvictionWaitSec, c.EvictionMaxWaitSec)
	}
	if c.EvictionGrace && c.IdleTimeoutSec > 0 && c.GraceSeconds() > c.IdleTimeoutSec {
		return fmt.Errorf(
			"the eviction grace (%d s) must not be longer than the idle timeout (%d s), "+
				"or the idle timeout unloads the model the wait is protecting",
			c.GraceSeconds(), c.IdleTimeoutSec)
	}
	// A maximum wait below the grace does not shorten the wait, it silently
	// disables the rule that stops one client starving another: a waiting
	// request may override a model's protection once its own age reaches the
	// grace, and it is refused once its age reaches the maximum, so with the
	// maximum the smaller of the two the first can never happen. Refused
	// rather than raised at a save because the two are one pair, edited
	// together in one fieldset, and telling the operator is better than
	// quietly serving them a different figure.
	if c.EvictionGrace && c.MaxWaitSeconds() < c.GraceSeconds() {
		return fmt.Errorf(
			"the maximum wait (%d s) must not be shorter than the eviction grace (%d s), "+
				"or a waiting request is refused before its own wait can override the grace",
			c.MaxWaitSeconds(), c.GraceSeconds())
	}
	return nil
}

// GraceSeconds and MaxWaitSeconds are the two intervals in force, with zero
// resolved to its default. Everything that acts on them reads them here, so
// "unset" has one meaning and not one per caller.
func (c Config) GraceSeconds() int {
	if c.EvictionGraceSec <= 0 {
		return DefaultEvictionGraceSec
	}
	return c.EvictionGraceSec
}

func (c Config) MaxWaitSeconds() int {
	if c.EvictionMaxWaitSec <= 0 {
		return DefaultEvictionMaxWaitSec
	}
	return c.EvictionMaxWaitSec
}

// sanitizeGrace repairs eviction-grace figures this build cannot use and
// returns what it repaired, so a hand-edited file, a backup or another build's
// settings still load.
//
// Repaired rather than refused, for the reason sanitizeStats gives: a refused
// config.json sends the next start into its fail-closed loopback-only branch.
// An out-of-range interval falls back to its default; a grace longer than the
// idle timeout is clamped to that timeout rather than dropped, because the
// operator asked for a grace and the longest one the reaper leaves intact is
// the timeout itself.
func (c *Config) sanitizeGrace() []string {
	var repaired []string
	if c.EvictionGraceSec < 0 || c.EvictionGraceSec > MaxEvictionWaitSec {
		repaired = append(repaired, "eviction_grace_sec="+strconv.Itoa(c.EvictionGraceSec))
		c.EvictionGraceSec = DefaultEvictionGraceSec
	}
	if c.EvictionMaxWaitSec < 0 || c.EvictionMaxWaitSec > MaxEvictionWaitSec {
		repaired = append(repaired, "eviction_max_wait_sec="+strconv.Itoa(c.EvictionMaxWaitSec))
		c.EvictionMaxWaitSec = DefaultEvictionMaxWaitSec
	}
	if c.EvictionGrace && c.IdleTimeoutSec > 0 && c.GraceSeconds() > c.IdleTimeoutSec {
		repaired = append(repaired, "eviction_grace_sec="+strconv.Itoa(c.GraceSeconds()))
		c.EvictionGraceSec = c.IdleTimeoutSec
	}
	// Raised rather than refused on this path, and raised after the clamp
	// above so it is measured against the grace that survives it. A file is
	// repaired; a save is told (see validateGrace).
	if c.EvictionGrace && c.MaxWaitSeconds() < c.GraceSeconds() {
		repaired = append(repaired, "eviction_max_wait_sec="+strconv.Itoa(c.MaxWaitSeconds()))
		c.EvictionMaxWaitSec = c.GraceSeconds()
	}
	return repaired
}

// Default returns the shipping defaults: LAN-exposed, unauthenticated.
func Default() Config {
	return Config{
		Host:                     "0.0.0.0",
		Port:                     11535,
		APIKey:                   "",
		UpstreamHeaderTimeoutSec: 0,
		Advertise:                true,
		IdleTimeoutSec:           0,
		DecodeConcurrency:        1,
		StatsMonths:              DefaultStatsMonths,
		StatsMaxBytes:            DefaultStatsMaxBytes,
		// Stored even though the feature is off, so that switching it on in
		// Settings is one tick rather than one tick and two numbers.
		EvictionGraceSec:   DefaultEvictionGraceSec,
		EvictionMaxWaitSec: DefaultEvictionMaxWaitSec,
	}
}

// The two levels Dessau writes its own log at. Sparse is one line per event
// that mattered; detailed adds the figures sparse omits. They are the strings
// config.json carries, the strings the control panel posts, and the strings
// docs/logging.md prints — one spelling, so a level cannot mean one thing in
// the panel and another in the file.
const (
	LogLevelSparse   = "sparse"
	LogLevelDetailed = "detailed"
)

// EffectiveLogLevel is the level in force: the one that was set, or sparse.
//
// Empty is the default rather than a level of its own, so a settings file
// written before this field existed, a field the operator cleared, and a fresh
// install all mean the same thing — the rule GraceSeconds and MaxWaitSeconds
// already follow for their own unset figures.
func (c Config) EffectiveLogLevel() string {
	if c.LogLevel == "" {
		return LogLevelSparse
	}
	return c.LogLevel
}

// SlogLevel is what the handler reads: sparse is Info and above, detailed is
// Debug and above.
//
// The mapping lives here, beside the names, because it is the whole of what
// the two words mean. Anything that turned a level name into a slog.Level
// somewhere else would be a second definition of "detailed", and the first
// time the two disagreed the panel would promise an operator figures the
// process was not writing.
func (c Config) SlogLevel() slog.Level {
	if c.EffectiveLogLevel() == LogLevelDetailed {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}

// validLogLevel reports whether name is one this build writes at. It is the
// one predicate both the repair and the refusal below are decided by.
func validLogLevel(name string) bool {
	return name == "" || name == LogLevelSparse || name == LogLevelDetailed
}

// sanitizeLogLevel repairs a log level this build cannot use and says that it
// did, so a hand-edited file, a backup or another build's settings still load.
//
// Repaired rather than refused, for the reason sanitizeStats gives: a refused
// config.json sends the next start into its fail-closed loopback-only branch,
// and a machine-wide outage is far too much to pay for a word that decides how
// much the log says. Reported rather than repaired silently, because the level
// IS in force in a changed form — an operator who wrote "verbose" and reads
// nothing would go looking in a detailed log that was never written. Save
// still refuses the same value outright, which is the moment the operator is
// there to read why.
func (c *Config) sanitizeLogLevel() []string {
	if validLogLevel(c.LogLevel) {
		return nil
	}
	repaired := []string{"log_level=" + c.LogLevel}
	c.LogLevel = ""
	return repaired
}

// The idle threshold's bounds: a minute is the shortest quiet a loop that
// ticks once a minute can tell from noise, and an hour is past the point where
// an idle job would never run on a Mac that is used at all.
const (
	DefaultIdleThresholdSec = 300
	MinIdleThresholdSec     = 60
	MaxIdleThresholdSec     = 3600
)

// EffectiveIdleThresholdSec is the idle threshold in force: the setting, or
// the default when none is set.
func (c Config) EffectiveIdleThresholdSec() int {
	if c.IdleThresholdSec == 0 {
		return DefaultIdleThresholdSec
	}
	return c.IdleThresholdSec
}

// EffectiveIdleThreshold is EffectiveIdleThresholdSec as a duration.
func (c Config) EffectiveIdleThreshold() time.Duration {
	return time.Duration(c.EffectiveIdleThresholdSec()) * time.Second
}

func usableIdleThreshold(sec int) bool {
	return sec == 0 || (sec >= MinIdleThresholdSec && sec <= MaxIdleThresholdSec)
}

// sanitizeIdleThreshold repairs a threshold this build cannot use and returns
// what it repaired, for the reason sanitizeStats gives.
func (c *Config) sanitizeIdleThreshold() []string {
	if usableIdleThreshold(c.IdleThresholdSec) {
		return nil
	}
	repaired := []string{"idle_threshold_sec=" + strconv.Itoa(c.IdleThresholdSec)}
	c.IdleThresholdSec = 0
	return repaired
}

// The update check's interval: an hour is often enough to see a fix the day
// it lands, and thirty days is the longest a check can be put off and still
// be one. Daily is what an operator who turns checks on gets unless they say.
const (
	DefaultUpdateCheckIntervalHours = 24
	MinUpdateCheckIntervalHours     = 1
	MaxUpdateCheckIntervalHours     = 720
)

// EffectiveUpdateCheckInterval is the update check's interval in force.
func (c Config) EffectiveUpdateCheckInterval() time.Duration {
	h := c.UpdateCheckIntervalHours
	if h == 0 {
		h = DefaultUpdateCheckIntervalHours
	}
	return time.Duration(h) * time.Hour
}

func usableUpdateCheckInterval(h int) bool {
	return h == 0 || (h >= MinUpdateCheckIntervalHours && h <= MaxUpdateCheckIntervalHours)
}

// sanitizeUpdateCheck repairs an interval this build cannot use and returns
// what it repaired, for the reason sanitizeStats gives. The switch is kept:
// an operator who turned checks on with a figure out of range wants checks.
func (c *Config) sanitizeUpdateCheck() []string {
	if usableUpdateCheckInterval(c.UpdateCheckIntervalHours) {
		return nil
	}
	repaired := []string{"update_check_interval_hours=" + strconv.Itoa(c.UpdateCheckIntervalHours)}
	c.UpdateCheckIntervalHours = 0
	return repaired
}

// sanitizeStats repairs a retention figure this build cannot use and returns
// what it repaired, so a hand-edited file, a backup or another build's
// settings still load.
//
// Repaired rather than refused, for the reason Load's own comment gives: a
// refused config.json sends the next start into its fail-closed loopback-only
// branch, and a machine-wide outage is far too much to pay for a number that
// decides how long a statistics file is kept. Save still refuses the same
// values outright, which is the moment the operator is there to read why.
func (c *Config) sanitizeStats() []string {
	var repaired []string
	if c.StatsMonths < 1 || c.StatsMonths > MaxStatsMonths {
		repaired = append(repaired, "stats_months="+strconv.Itoa(c.StatsMonths))
		c.StatsMonths = DefaultStatsMonths
	}
	if c.StatsMaxBytes < MinStatsMaxBytes || c.StatsMaxBytes > MaxStatsMaxBytes {
		repaired = append(repaired, "stats_max_bytes="+strconv.FormatInt(c.StatsMaxBytes, 10))
		c.StatsMaxBytes = DefaultStatsMaxBytes
	}
	return repaired
}

// Validate reports whether the config is usable.
func (c Config) Validate() error {
	if c.Port < 1 || c.Port > 65535 {
		return fmt.Errorf("port %d out of range", c.Port)
	}
	if c.Host == "" {
		return errors.New("host must not be empty")
	}
	// A Host that cannot be bound is refused here rather than at the listener.
	// It used to travel two ways: cmd/dessau built "<host>:<port>" and the
	// process exited when that would not listen, and — before it got that far —
	// gateway.Endpoints pasted the same value into the base URL the panel, the
	// menu bar and the clipboard hand out. A value carrying CR or LF in a base
	// URL is a header-injection primitive in whichever client takes it, so the
	// fence belongs where the value is first read rather than at each surface
	// that repeats it. Load turns a refusal here into a lock-down to loopback,
	// which is the same fail-closed path a corrupt file takes.
	if !ValidBindHost(c.Host) {
		return fmt.Errorf("host %q is neither an IP address nor a host name", c.Host)
	}
	// An unrecognised bind mode decides the bind, and nothing downstream knows
	// what it decided. Refused rather than repaired, in the direction every
	// other bind fault runs: Load turns this into a loopback bind with the
	// rest of the operator's settings kept, and Save turns it into a message
	// while they are there to read it.
	if c.BindMode != BindModeHost && c.BindMode != BindModePrivateNetwork {
		return fmt.Errorf("bind_mode %q is not a bind mode this build has; leave it out for the address in \"host\", or set %q", c.BindMode, BindModePrivateNetwork)
	}
	if c.DecodeConcurrency < 1 {
		return fmt.Errorf("decode_concurrency must be >= 1, got %d", c.DecodeConcurrency)
	}
	if !usableIdleThreshold(c.IdleThresholdSec) {
		return fmt.Errorf("idle_threshold_sec must be between %d and %d, or 0 for the default, got %d",
			MinIdleThresholdSec, MaxIdleThresholdSec, c.IdleThresholdSec)
	}
	if !usableUpdateCheckInterval(c.UpdateCheckIntervalHours) {
		return fmt.Errorf("update_check_interval_hours must be between %d and %d, or 0 for daily, got %d",
			MinUpdateCheckIntervalHours, MaxUpdateCheckIntervalHours, c.UpdateCheckIntervalHours)
	}
	// Named fields, both of them: this is the settings path, where a person is
	// waiting to be told which of a form's worth of settings was refused.
	if len(c.APIKey) > MaxAPIKeyBytes {
		return fmt.Errorf("api_key must be at most %d bytes, got %d", MaxAPIKeyBytes, len(c.APIKey))
	}
	if len(c.Preload) > MaxPreload {
		return fmt.Errorf("preload names %d models, more than the %d this holds", len(c.Preload), MaxPreload)
	}
	// A TLS port this Mac cannot listen on. Refused here and REPAIRED at load
	// (sanitizeTLSPort), which is the same two-sided treatment log_level gets:
	// a hand-edited file is narrowed rather than locked out, and a save that
	// actually carries the value is told while the operator is there to read
	// it. NoTLSListener is the one value below 1 that means something.
	if c.TLSPort != NoTLSListener && (c.TLSPort < 0 || c.TLSPort > 65535) {
		return fmt.Errorf("tls_port %d is not a port; use 0 for the port beside \"port\", or %d for no TLS listener", c.TLSPort, NoTLSListener)
	}
	if c.TLSPort == c.Port {
		return fmt.Errorf("tls_port %d is the port this server already answers on; use 0 for the port beside it, or %d for no TLS listener", c.TLSPort, NoTLSListener)
	}
	if err := c.validateClients(); err != nil {
		return err
	}
	if err := c.validateModels(); err != nil {
		return err
	}
	if err := c.ChatRule.validate(); err != nil {
		return err
	}
	// Eviction grace on an open endpoint is a denial-of-service lever: a parked
	// waiter that cannot progress blocks every cold load needing an eviction
	// for as long as the maximum wait allows, and the queue is shared out per
	// API key — which means it cannot be shared out at all when there is no key
	// to tell callers apart. Loopback-only installs are unaffected: there is no
	// network caller to defend against, and the check turns on exposure rather
	// than on the key alone.
	// The private-network mode counts as exposed HERE and only here. This runs
	// on a stored configuration — at a save, and at a load before any socket
	// exists — so nothing can say what the mode would bind, and "might serve
	// network callers" is the strongest thing that can be asked of it. Erring
	// closed costs a key on a feature that is off by default; erring open
	// costs the queue this rule protects.
	if c.EvictionGrace && c.APIKey == "" {
		// Two spellings of one rule, because the operator has to recognise the
		// server being described. A loopback Host under the private-network
		// mode is not a LAN-exposed server, and telling them it is sends them
		// to look at a bind address that is not what fired this.
		const why = ": without one the wait queue cannot be shared out between callers, and one client can hold up model loading for everyone"
		switch {
		case c.ExposedToLAN():
			return errors.New("eviction grace needs an API key on a LAN-exposed server" + why)
		case c.BindMode == BindModePrivateNetwork:
			return errors.New("eviction grace needs an API key on a server that may reach other machines (private-network mode)" + why)
		}
	}
	if c.StatsMonths < 1 || c.StatsMonths > MaxStatsMonths {
		return fmt.Errorf("keep statistics for between 1 and %d months, got %d", MaxStatsMonths, c.StatsMonths)
	}
	if c.StatsMaxBytes < MinStatsMaxBytes || c.StatsMaxBytes > MaxStatsMaxBytes {
		return fmt.Errorf("the statistics store's limit must be between %d and %d bytes, got %d",
			MinStatsMaxBytes, MaxStatsMaxBytes, c.StatsMaxBytes)
	}
	// The one enum in this file that is repaired on the way in and refused
	// here: Load sanitizes before it validates, so a hand-edited word never
	// reaches this check, and what does reach it is a save the operator is
	// standing in front of.
	if !validLogLevel(c.LogLevel) {
		return fmt.Errorf("log_level %q is not one this build writes at; use %q or %q",
			c.LogLevel, LogLevelSparse, LogLevelDetailed)
	}
	if c.MaxResidentBytes < 0 {
		return fmt.Errorf("max_resident_bytes must not be negative, got %d", c.MaxResidentBytes)
	}
	if err := c.validateGrace(); err != nil {
		return err
	}
	// A sampling default becomes a launch flag on every model server, and the
	// model server validates the effective value of every request against it:
	// a value it rejects turns one save into a 400 on every request that omits
	// that parameter. Load sanitizes before it validates, so this strictness
	// only ever refuses a save, never a start-up.
	return c.validateSampling()
}

// The two bind modes. A third choice in Settings, never automatic: switching
// it on changes who can reach an existing install, and a default that moves
// the day someone installs a VPN is a default change wearing a feature's
// clothes.
const (
	// BindModeHost is the default: Config.Host is the bind.
	BindModeHost = ""
	// BindModePrivateNetwork serves on the one address this Mac holds on a
	// private network, and on this Mac. It fails closed to this Mac alone when
	// there is no such address, or more than one — it never picks between them
	// (adr-2609081118587999, amendment condition 1).
	BindModePrivateNetwork = "private-network"
)

// ValidBindHost reports whether a value is something cmd/dessau can bind.
//
// It is as wide as the listener and no wider, and that is checked rather than
// asserted: every value the table in host_test.go marks bindable was watched
// to produce a listener, and every value it refuses was watched to fail.
//
// An IPv6 literal is accepted in either spelling. The listen address is built
// by internal/bind through net.JoinHostPort (adr-2609091123526871 rule 5),
// which brackets a host carrying colons itself, so "::1" and "[::1]" name the
// same bind and both listen. That was not true while the address was built
// with fmt.Sprintf: "::1:11535" came back from net.SplitHostPort as "too many
// colons in address" and the app exited before it served anything, which is
// iss-7's second fault. The bracketed spelling stays accepted because
// config.json files carry it.
//
// What a colon still cannot do is carry a port. "192.168.1.5:8080" parses as
// no address, and a colon is not legal in a host-name label, so it is refused
// here rather than bracketed by JoinHostPort into an address no listener
// takes. A zone ("fe80::1%en0", bracketed or not) binds and stays legal, even
// though URLHost refuses to put one in a URL.
func ValidBindHost(host string) bool {
	bare, ok := unbracket(host)
	if !ok {
		return false
	}
	if addr, zone, hasZone := strings.Cut(bare, "%"); hasZone {
		return net.ParseIP(addr) != nil && validHostLabel(zone)
	}
	return net.ParseIP(bare) != nil || validHostName(bare)
}

// URLHost returns the host as a URL must spell it, and reports whether it can
// appear in one at all.
//
// The brackets a bind needs come off exactly once here: net.JoinHostPort adds
// its own, and passing it a host that is already bracketed produced
// "http://[[::1]]:11535/v1" — the address of nothing, handed out as the base
// URL of everything. A zone is refused rather than carried: the "%" that
// separates it is an escape introducer in a URL and not a literal, so there is
// no spelling of "fe80::1%en0" that both means what it says and parses.
// Refusing leaves that address off the list, which is the same answer the list
// gives for every other address it cannot describe truthfully.
func URLHost(host string) (string, bool) {
	bare, ok := unbracket(host)
	if !ok || strings.Contains(bare, "%") {
		return "", false
	}
	if net.ParseIP(bare) == nil && !validHostName(bare) {
		return "", false
	}
	return bare, true
}

// unbracket removes the brackets an IPv6 bind is written with, and refuses a
// value that is bracketed on one side only or bracketed around nothing.
func unbracket(host string) (string, bool) {
	if host == "" {
		return "", false
	}
	opened, closed := strings.HasPrefix(host, "["), strings.HasSuffix(host, "]")
	switch {
	case opened && closed:
		inner := host[1 : len(host)-1]
		if inner == "" || strings.ContainsAny(inner, "[]") {
			return "", false
		}
		return inner, true
	case opened || closed:
		return "", false
	}
	return host, !strings.ContainsAny(host, "[]")
}

// validHostName reports whether a value is a host name: dot-separated labels,
// optionally fully qualified with a trailing dot. Underscores are allowed
// inside a label — they are not RFC 1123, and they are handed out by real
// networks, and refusing one here would stop a server that binds today.
//
// A name is refused when it is really an address, and that takes two rules
// rather than one, because RFC 1123 §2.1's "the top label is alphabetic" is
// not the same test as "contains a letter" — which is what this used to
// apply, and hex spells an address with letters in it. Measured, every one of
// "0x0", "0X0", "0x00000000", "0x0.0x0.0x0.0x0" and "0.0.0.0x0" was accepted
// and bound "[::]", every interface on this Mac, while "0x7f000001",
// "0x7f.0x0.0x0.0x1" and "127.0.0.0x1" were accepted and bound 127.0.0.1.
//
// So: the top label must carry a letter, AND the labels must not all be
// numeric in one of the forms inet_aton reads. Measured on this platform,
// net.Listen takes "0:0" and returns a listener on "[::]" — the unspecified
// address, every interface on the machine — while "127.1", "2130706433",
// "0x7f.1" and "0177.0.0.1" all come back 127.0.0.1.
//
// Both directions are wrong, and in opposite ways. `{"host":"0"}` bound
// everything while everything downstream read it as a specific bind by name,
// so the control panel offered "http://0:11535/v1" and listed nothing else: a
// wildcard under-reported, which is the dead-address fault seen from the other
// side. `{"host":"127.1"}` bound loopback while ExposedToLAN read a name and
// told the operator they bind a LAN address.
//
// Refusing is the closed direction: Validate refuses the file, and cmd/dessau
// locks the bind down to loopback rather than binding wider than the panel
// says. Nobody writes "0" meaning the wildcard; they write "0.0.0.0" or leave
// it empty, and both still work.
func validHostName(s string) bool {
	if len(s) > 253 {
		return false
	}
	s = strings.TrimSuffix(s, ".")
	if s == "" {
		return false
	}
	labels := strings.Split(s, ".")
	numeric := true
	for _, label := range labels {
		if !validHostLabel(label) {
			return false
		}
		if !numericLabel(label) {
			numeric = false
		}
	}
	if numeric {
		return false
	}
	return hasLetter(labels[len(labels)-1])
}

// numericLabel reports whether a label is one of the numeric forms inet_aton
// reads a part of an address as: decimal, octal written with a leading zero,
// or hex written with a leading "0x". A value whose every label is one of
// these is an address however many parts it has, so it is not a name, whatever
// letters the hex spelling happens to contain.
//
// It is deliberately this grammar rather than strconv.ParseUint(label, 0, 64),
// which also reads Go's own "0b"/"0o" prefixes and digit-separating
// underscores — spellings getaddrinfo does not accept, so a value Go's parser
// calls numeric is not necessarily an address this platform would resolve as
// one. Borrowing it would decide the question by a different language's
// literal syntax than the one the resolver speaks.
//
// The earlier version of this comment justified that with "1_0.2_0", a name it
// claimed resolves perfectly well and Go's parser would refuse. That example
// is wrong about this code: validHostName refuses "1_0.2_0" above, on the rule
// that the top label must carry a letter, and refused it before this grammar
// existed. The reasoning stands; the example never did.
func numericLabel(s string) bool {
	if s == "" {
		return false
	}
	if len(s) > 2 && s[0] == '0' && (s[1] == 'x' || s[1] == 'X') {
		for i := 2; i < len(s); i++ {
			c := s[i]
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
		return true
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// hasLetter reports whether a label contains an ASCII letter. It is what
// separates a host name's top label from a legacy spelling of an IPv4 address.
func hasLetter(label string) bool {
	for i := 0; i < len(label); i++ {
		if c := label[i]; (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') {
			return true
		}
	}
	return false
}

// validHostLabel reports whether one dot-separated label is well formed. It is
// also what an IPv6 zone is held to: an interface name, which on this platform
// is letters and digits.
func validHostLabel(label string) bool {
	if label == "" || len(label) > 63 {
		return false
	}
	for i := 0; i < len(label); i++ {
		c := label[i]
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '_':
		case c == '-' && i != 0 && i != len(label)-1:
		default:
			return false
		}
	}
	return true
}

// ExposedToLAN reports whether the bind address accepts non-loopback traffic.
// Anything that is not loopback counts: a specific interface address exposes
// the gateway to the LAN just as the wildcard does, and must trigger the same
// security warnings.
//
// It is read by everything that decides how open this server is — the
// generate-a-key-or-drop-to-loopback branch in cmd/dessau, the eviction-grace
// key requirement, the panel's warning, whether Bonjour advertises at all, and
// whether the endpoint list enumerates this machine's addresses — so it has to
// understand the same spellings of the bind that the listener does. It did
// not: it compared c.Host as a string, and an IPv6 literal binds only when
// config.json carries it bracketed, so a "[::1]" bind was read as LAN-exposed.
// That errs closed — a key was generated for a server nothing off this Mac can
// reach — but it also told the operator "this server binds a LAN address" and
// advertised a Bonjour service no machine on the LAN could connect to, which
// is a false statement about their exposure (adr-2609081118587999 rule 4).
//
// Everything this cannot resolve to a loopback address is exposed. That is the
// direction the errors have to run: a name resolves to whatever the resolver
// says today, and a malformed value binds nothing at all, and neither is a
// reason to stand down.
//
// What it does NOT answer is what the running server is exposed on. A bind is
// a set of addresses now, the set can be narrower than the configuration asked
// for, and the bind mode may name no address at all — so everything that
// decides at startup or reports at runtime asks bind.Plan.ReachesOtherMachines
// instead, which is a question about sockets (adr-2609091123526871 rule 7).
// This is the answer about a stored configuration, which is what a stored
// configuration can be asked, and Validate below is its remaining reader.
func (c Config) ExposedToLAN() bool {
	bare, ok := unbracket(c.Host)
	if !ok {
		return true
	}
	// A zone belongs to the interface, not to the address: "[::1%lo0]" is the
	// same loopback bind as "[::1]".
	if addr, _, hasZone := strings.Cut(bare, "%"); hasZone {
		bare = addr
	}
	// The resolver folds case and ignores a trailing dot, and this compared
	// bytes: "LOCALHOST", "LocalHost" and "localhost." each bind 127.0.0.1 and
	// nothing else, and each was read here as a LAN bind. Same class as the
	// "[::1]" fault above, and the same four false statements follow from it.
	if strings.EqualFold(strings.TrimSuffix(bare, "."), "localhost") {
		return false
	}
	if ip := net.ParseIP(bare); ip != nil {
		return !ip.IsLoopback()
	}
	return true
}

// Load reads config from path, returning defaults if the file does not exist.
// Unknown or missing fields fall back to their defaults, so a config written by
// an older build still loads.
//
// The second return value names the sampling preferences that were dropped
// because the model server would not accept them, or because they name no
// addressable model; the caller logs them. They are dropped rather than
// refused because a sampling value is a preference, not a serving invariant:
// refusing the file sends main into its fail-closed loopback-only mode, which
// is a machine-wide outage to pay for one number that could simply be
// ignored. Everything that decides how the server is reachable is still
// validated, and still an error.
//
// This covers a value out of range, not a value of the wrong shape. A
// sampling field holding a string, or a number too large for a float64, fails
// in the unmarshal above and is an error like any other malformed config.json
// — sampling is not special enough to warrant a second decoding path through
// json.RawMessage in a file edited by hand.
//
// The read is hardened (see OpenRegular): Load runs before the port is
// claimed, so a FIFO under this name would otherwise
// hang startup before the fail-closed branch in main could ever run, and a
// symlinked or oversized file is refused rather than applied. Any such refusal
// is an error, which main treats as "lock down to loopback".
func Load(path string) (Config, Notices, error) {
	cfg := Default()
	b, err := ReadRegular(path, MaxConfigBytes)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, Notices{}, nil
	}
	if err != nil {
		return cfg, Notices{}, fmt.Errorf("read config: %w", err)
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Default(), Notices{}, fmt.Errorf("parse config %s: %w", path, err)
	}
	var n Notices
	// Ignored: the setting is not in force at all, and setting it again is the
	// only way to get it.
	n.Ignored = append(n.Ignored, SupersededSettings(b)...)
	n.Ignored = append(n.Ignored, cfg.sanitizeSampling()...)
	n.Ignored = append(n.Ignored, cfg.sanitizeModels()...)
	n.Ignored = append(n.Ignored, cfg.sanitizePreload()...)
	n.Ignored = append(n.Ignored, cfg.sanitizeClients()...)
	// Repaired: the setting IS in force, in a changed form. Telling an
	// operator to set it again would send them looking for a value that is
	// working — and for the API key it would be worse than that, because the
	// trimmed key is the one their clients must now send.
	n.Repaired = append(n.Repaired, cfg.sanitizeStats()...)
	n.Repaired = append(n.Repaired, cfg.sanitizeIdleThreshold()...)
	n.Repaired = append(n.Repaired, cfg.sanitizeUpdateCheck()...)
	n.Repaired = append(n.Repaired, cfg.sanitizeBudget()...)
	n.Repaired = append(n.Repaired, cfg.sanitizeGrace()...)
	n.Repaired = append(n.Repaired, cfg.sanitizeAPIKey()...)
	n.Repaired = append(n.Repaired, cfg.sanitizeLogLevel()...)
	n.Repaired = append(n.Repaired, cfg.sanitizeChatRule()...)
	n.Repaired = append(n.Repaired, cfg.sanitizeTLSPort()...)
	if err := cfg.Validate(); err != nil {
		return Default(), Notices{}, &InvalidError{Path: path, Err: err, Parsed: cfg, Notices: n}
	}
	return cfg, n, nil
}

// superseded names the settings keys this build no longer reads, and what
// carries each of them now.
//
// Every one of them was a per-model setting with a shape of its own. They are
// one map today (iss-2609062213413447), and pre-1.0 that migration is made by
// the operator rather than by a compatibility path nobody would ever be able
// to delete: the old keys are not read, and the next save writes the new shape.
var superseded = map[string]string{
	"model_sampling": "replaced by models[<id>].sampling",
	"per_model":      "replaced by models[<id>]",
	"pinned":         "replaced by models[<id>].pinned",
}

// SupersededSettings names the superseded keys a settings body or file still
// carries, each with what carries it now.
//
// One function for both surfaces, because they owe the same answer. On the
// file path Load reports them as ignored, so an operator is told once — at the
// start that ignored them — rather than left to wonder why a model is no
// longer pinned. On the settings path the endpoint refuses the body outright:
// a caller posting one of these keys is a script or a shell of someone's own,
// and answering "saved" to a save that changed nothing is the one reply that
// leaves them believing it worked.
//
// Read off the raw bytes, because the fields are gone from the type and
// encoding/json says nothing about a key it does not know. Matched the way
// encoding/json matches a field name, case-insensitively, so a hand-edited
// "Pinned" is reported rather than silently dropped.
func SupersededSettings(b []byte) []string {
	var named map[string]json.RawMessage
	if err := json.Unmarshal(b, &named); err != nil {
		return nil
	}
	var out []string
	for _, key := range slices.Sorted(maps.Keys(superseded)) {
		for k := range named {
			if strings.EqualFold(k, key) {
				out = append(out, key+" ("+superseded[key]+")")
				break
			}
		}
	}
	return out
}

// Notices is what Load had to change about a settings file to make it usable,
// kept in two lists because the difference is the whole of what an operator
// needs to hear.
//
// A setting in Ignored is not in force at all: it named nothing this build can
// use, and setting it again is the only way to get it. A setting in Repaired
// IS in force, in a changed form — trimmed, clamped, or replaced by the
// default that stands behind it. One message for both said "ignoring settings
// the model server would not accept — set them again", which is untrue of
// every repair and dangerous for exactly one of them: a trimmed API key is the
// key clients must send from that moment on, and an operator told it was
// ignored has been told the opposite of what happened.
type Notices struct {
	Ignored  []string
	Repaired []string
}

// Empty reports whether the file needed no changing at all.
func (n Notices) Empty() bool { return len(n.Ignored) == 0 && len(n.Repaired) == 0 }

// All names everything Load changed, ignored and repaired together, for a
// caller that wants the fields and not the distinction.
func (n Notices) All() []string {
	out := make([]string, 0, len(n.Ignored)+len(n.Repaired))
	out = append(out, n.Ignored...)
	return append(out, n.Repaired...)
}

// InvalidError reports a config.json that read and parsed cleanly and then
// failed Validate, and carries the configuration it parsed.
//
// The distinction is the whole point of the type. A file that cannot be read
// or cannot be parsed tells us nothing about what the operator wanted, and the
// only safe answer is the shipping defaults with the bind locked down. A file
// that parsed tells us everything except the one field Validate objected to —
// and the caller was throwing all of it away: an API key, a port, pinned
// models, a memory budget and a statistics retention period were silently
// replaced by defaults because a hand-edited Host would not bind, under a log
// line reading "config.json could not be read" about a file that read fine.
//
// The first return value of Load stays Default() so a caller that ignores the
// error is unchanged. A caller that handles it can narrow the lockdown to the
// bind.
type InvalidError struct {
	Path string
	Err  error
	// Parsed is the configuration as it was read: sanitized, and invalid in
	// whatever way Err names. It is not safe to run as it stands.
	Parsed Config
	// Notices names what sanitizing changed on the way here, split the way
	// Load's second return value would have carried it: settings that are not
	// in force at all, and settings that are in force in a changed form.
	Notices Notices
}

func (e *InvalidError) Error() string { return "invalid config " + e.Path + ": " + e.Err.Error() }

func (e *InvalidError) Unwrap() error { return e.Err }

// Save atomically writes config to path.
//
// A config that would not load again is refused rather than written. Load caps
// what it will read, and a config.json over that cap is not a smaller problem
// than a corrupt one: main falls back to loopback-only with the shipping
// defaults, so the API key and the bind address a user set are silently
// unused until someone edits the file by hand. The check is here rather than
// in Validate because it is a property of the encoded bytes — MarshalIndent's
// output is larger than the body it came from, so bounding the request that
// carried it is not enough.
func Save(path string, c Config) error {
	if err := c.Validate(); err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	if len(b)+1 > MaxConfigBytes {
		return fmt.Errorf("settings are %d bytes, over the %d-byte limit config.json can be read back from",
			len(b)+1, MaxConfigBytes)
	}
	return writeSettingsFile(path, append(b, '\n'))
}

// writeSettingsFile writes a settings file atomically and closed (0600). It is
// the one writer of config.json and of the server's TLS private key
// (WriteSecretFile), so the hardening below is stated once and cannot drift.
func writeSettingsFile(path string, b []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	// config.json holds the API key and HuggingFace token. Never write it through
	// a predictable temp name: a "config.json.tmp" created ahead of the write as
	// a symlink would redirect it, and one with loose permissions would keep
	// them through the rename. os.CreateTemp uses a random name with O_EXCL and
	// mode 0600, closing both holes.
	tmp, err := os.CreateTemp(dir, "config-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) // harmless no-op once the rename succeeds
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return err
	}
	// Flush to disk before the rename. The rename is atomic against a process
	// crash, but not against a power loss that makes the rename durable before the
	// data — which would leave a truncated config.json. A truncated config fails to
	// parse and reverts to defaults, so durability here is a security concern, not
	// just a tidiness one.
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

// privateToThisAccount reports whether a secret file is one only this
// account could have written, and says why not when it is not. It takes three
// facts from the fstat of the handle the bytes came from — never a second stat
// of the path, which could be raced.
//
// Ownership alone is not the boundary:
//
//   - A hard link. A file this account owns can be linked under the name being
//     read by anything that can write the directory, and the uid then reads as
//     ours while the content is whatever the linked file holds. A file written
//     through writeSettingsFile has exactly one link, so more than one means
//     something else made it.
//   - A loose mode. A secret file left group- or world-writable is one another
//     account could have written before this start read it. Anything outside
//     0600 is refused.
func privateToThisAccount(info os.FileInfo) error {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return errors.New("cannot read the file's ownership on this platform")
	}
	if int(st.Uid) != os.Getuid() {
		return fmt.Errorf("it belongs to another account (uid %d)", st.Uid)
	}
	if st.Nlink != 1 {
		return fmt.Errorf("it has %d hard links, so another account may have linked it here", st.Nlink)
	}
	if perm := info.Mode().Perm(); perm&0o077 != 0 {
		return fmt.Errorf("its mode is %#o, so another account could have read or written it", perm)
	}
	return nil
}

// GenerateAPIKey returns a fresh random API key, 32 bytes of crypto/rand
// rendered as URL-safe base64 without padding.
//
// Used to fail closed rather than open: a server that binds a LAN address with
// no key configured is reachable, unauthenticated, by everyone on the network,
// and the warning that said so was the only thing standing between a fresh
// install and an open endpoint. A generated key is announced loudly, persisted,
// and shown in the control panel, so the operator can use it or replace it —
// but there is no window in which the endpoint is open by default.
func GenerateAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Challenge coordination: how a starting Dessau proves the process already on
// its port shares its data root, without either side keeping a secret.
//
// The prober writes a random answer into a random-named file in the data root
// and asks the holder, over loopback, to read that file back. Only a process
// that can read this root can answer, which is exactly the claim being tested.
// Nothing is stored between probes and nothing replayable crosses the wire: the
// file is single-use and deleted, so a peer who observes one answer learns
// nothing about the next.
//
// This replaces a durable per-run token, which failed in both directions. It
// could not authenticate a peer account (each root holds a different token), and
// it was replayable: the token was published to every loopback caller, survived
// shutdown on disk, and was read by the next start before a fresh one was
// written — so a local account could harvest it, wait, squat the port, and have
// the real server adopt it as its own.
const (
	// ChallengeFilePrefix names a challenge file. The leading dot keeps it out
	// of ordinary listings; the name after it is the caller's nonce.
	ChallengeFilePrefix = ".dessau-challenge-"
	// MaxChallengeBytes caps the answer read. A real answer is 64 hex chars.
	MaxChallengeBytes = 4096
	// challengeNameLen is the nonce length in hex characters (16 random bytes).
	challengeNameLen = 32
)

// ValidChallengeName reports whether name is a well-formed nonce.
//
// This is a path-traversal guard, not a formatting nicety: the name is supplied
// by the caller and used to build a path the server then READS. Without it,
// "../../../etc/passwd" would turn the control plane into an arbitrary-file-read
// oracle for anything the server's uid can open. Exactly 32 lowercase hex
// characters admits no separator, no dot, and no escape.
func ValidChallengeName(name string) bool {
	if len(name) != challengeNameLen {
		return false
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// ChallengePath returns the file a challenge name refers to under root, or ""
// when the name is not well-formed. Callers must treat "" as a refusal: it is
// the single point where an untrusted name is turned into a path.
func ChallengePath(root, name string) string {
	if !ValidChallengeName(name) {
		return ""
	}
	return filepath.Join(root, ChallengeFilePrefix+name)
}

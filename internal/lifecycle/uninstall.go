package lifecycle

import (
	"io/fs"
	"os"
	"path/filepath"
	"strconv"

	"github.com/intentdriven/Dessau/internal/config"
)

// `dessau uninstall` removes what this installation put on the Mac, and
// nothing else.
//
// EVERY DELETION PATH IS FIXED. Not one of them comes from a flag or from the
// environment: DESSAU_ROOT is read only so the output can say which root was
// NOT removed. A data root a caller can name is a data root any local account
// can pre-create as a symlink, and it would then choose what is deleted.
//
// NOTHING ELEVATES AND NOTHING SHELLS OUT. Every removal is os.RemoveAll in
// this process, acting with this account's own rights: a file this account
// cannot delete is reported, never elevated for. The single exception is the
// firewall entry, which is machine-wide state with no per-account route, and it
// goes through the same one authorisation panel as the grant — a refusal leaves
// everything else removed and reports the entry as what remains.
//
// THE MODELS STAY. They are the expensive thing to fetch again, so removing
// them is a separate decision with its own flag.

// UninstallEnv is everything uninstall acts on, resolved before anything is
// removed so the plan can be read in one place.
type UninstallEnv struct {
	Paths config.Paths
	Home  string
	// Bundles are the fixed locations an installed bundle can be at: the
	// machine-wide applications directory and this account's own. Both are in
	// play and they are not symmetric — one bundle in the machine-wide
	// directory is what every account on the Mac launches.
	Bundles []string
	// SystemApplications is that machine-wide directory, so the output can say
	// when what it removed was everybody's copy rather than this account's.
	SystemApplications string
	// Link is this account's own dessau command.
	Link string
	// Binary is the path the firewall entry is keyed to.
	Binary string
	// NamedRoot is what DESSAU_ROOT names, so the output can say it was not
	// acted on. It is never a deletion path.
	NamedRoot string
	// Terminal says whether standard input is a terminal, which is the only
	// thing that could answer a question — and nothing here reads it either
	// way.
	Terminal bool
	// Firewall removes the entry, raising the one authorisation panel.
	Firewall func(binary string) error
}

// RunUninstall is the uninstall verb.
func RunUninstall(env Env, args []string) int {
	ue, err := liveUninstallEnv(env)
	if err != nil {
		writeLine(env.Err, "dessau uninstall: "+err.Error())
		return ExitFailed
	}
	return runUninstall(env, args, ue)
}

// liveUninstallEnv resolves what a real run acts on, from the fixed locations
// and from nothing else.
func liveUninstallEnv(env Env) (UninstallEnv, error) {
	if err := liveEnvGuard("uninstall"); err != nil {
		return UninstallEnv{}, err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return UninstallEnv{}, err
	}
	root, err := config.InstalledRoot()
	if err != nil {
		return UninstallEnv{}, err
	}
	paths := config.NewPaths(root)
	bundles := []string{
		filepath.Join(systemApplications, bundleName),
		filepath.Join(home, "Applications", bundleName),
	}
	binary := filepath.Join(bundles[1], binaryInBundle)
	for _, b := range bundles {
		if _, err := os.Lstat(b); err == nil {
			binary = filepath.Join(b, binaryInBundle)
			break
		}
	}
	return UninstallEnv{
		Paths:              paths,
		Home:               home,
		Bundles:            bundles,
		SystemApplications: systemApplications,
		Link:               filepath.Join(binDir(home), linkFileName),
		Binary:             binary,
		NamedRoot:          os.Getenv("DESSAU_ROOT"),
		Terminal:           isTerminal(os.Stdin),
		Firewall:           revokeFirewall,
	}, nil
}

func runUninstall(env Env, args []string, ue UninstallEnv) int {
	fs := flags("uninstall", env.Err)
	purge := fs.Bool("purge", false, "remove the downloaded models as well")
	yes := fs.Bool("yes", false, "answer the confirmation --purge would otherwise need a terminal for")
	if err := fs.Parse(args); err != nil {
		return ExitUsage
	}
	if fs.NArg() > 0 {
		writeLine(env.Err, "dessau uninstall: unexpected argument "+Quote(fs.Arg(0)))
		return ExitUsage
	}

	// The consent check comes first, and it deletes nothing before it. Nothing
	// reads standard input: under a piped bootstrap that is the rest of the
	// installer, so the refusal names the flag that would have answered it.
	if *purge && !ue.Terminal && !*yes {
		writeLine(env.Err, "dessau uninstall --purge deletes the downloaded models, and standard input is not a terminal, "+
			"so there is nobody to confirm it with. Nothing has been deleted.")
		writeLine(env.Err, "Pass --yes to confirm it without a terminal: dessau uninstall --purge --yes")
		return ExitUsage
	}

	failed := false
	var remaining []string

	// The firewall entry first, while the binary it is keyed to still exists.
	if err := ue.Firewall(ue.Binary); err != nil {
		remaining = append(remaining,
			"the firewall entry for "+redact(ue.Binary, ue.Home)+" ("+err.Error()+")\n"+
				"    remove it with: "+firewallRemoveCommand(ue.Binary, ue.Home))
	} else {
		writeLine(env.Out, "removed the firewall entry.")
	}

	for _, target := range removalTargets(ue) {
		if _, err := os.Lstat(target); err != nil {
			continue
		}
		if err := os.RemoveAll(target); err != nil {
			// Never elevated for, and never handed to an external removal
			// command: a path this account cannot delete is reported as what it
			// is.
			remaining = append(remaining, redact(target, ue.Home)+" ("+err.Error()+")")
			failed = true
			continue
		}
		line := "removed " + redact(target, ue.Home) + "."
		if isUnder(target, ue.SystemApplications) {
			// Said rather than left to be discovered. One bundle there is what
			// every account on this Mac launches, so this removal was not only
			// this account's — and nobody else gets a message about it.
			line += " That was the copy every account on this Mac launches."
		}
		writeLine(env.Out, line)
	}

	if ok := removeLink(ue); ok {
		writeLine(env.Out, "removed the "+linkFileName+" command.")
	}

	reportModels(env, ue, *purge, &failed)

	if len(remaining) > 0 {
		writeLine(env.Out, "")
		writeLine(env.Out, "What is left:")
		for _, r := range remaining {
			writeLine(env.Out, "  - "+r)
		}
	}
	if ue.NamedRoot != "" {
		writeLine(env.Out, "")
		writeLine(env.Out, "DESSAU_ROOT names "+redact(ue.NamedRoot, ue.Home)+
			". Uninstall never derives a deletion path from the environment, so nothing there was removed.")
	}
	if failed {
		return ExitFailed
	}
	return ExitOK
}

// isUnder reports whether a path sits directly inside a directory. Empty dir
// means the question does not apply, which is how a test says "not here".
func isUnder(path, dir string) bool {
	return dir != "" && filepath.Dir(path) == filepath.Clean(dir)
}

// removalTargets is what uninstall removes, in the order it removes it. Every
// entry is derived from the resolved layout; none of them is a path a caller
// named.
//
// The data root itself is NOT on the list even on a per-user install, because
// the models live inside it: what goes is each thing this installation put
// there, by name.
func removalTargets(ue UninstallEnv) []string {
	targets := append([]string{}, ue.Bundles...)
	return append(targets,
		ue.Paths.Venv,
		ue.Paths.Python,
		ue.Paths.Bin,
		ue.Paths.Config,
		ue.Paths.State,
		ue.Paths.Logs,
		ue.Paths.Stats,
	)
}

// removeLink removes this account's dessau command, and only when it is a
// symbolic link to a binary inside a Dessau bundle. A real file of that name
// belongs to whoever put it there.
func removeLink(ue UninstallEnv) bool {
	if ue.Link == "" {
		return false
	}
	fi, err := os.Lstat(ue.Link)
	if err != nil || fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(ue.Link)
	if err != nil || filepath.Base(filepath.Dir(target)) != "MacOS" {
		return false
	}
	return os.Remove(ue.Link) == nil
}

// reportModels says what happens to the downloaded models: what was left and
// how much of it, or — under --purge — what was removed and what could not be.
func reportModels(env Env, ue UninstallEnv, purge bool, failed *bool) {
	dirs := []string{ue.Paths.Models, ue.Paths.HFCache}

	if !purge {
		size := 0
		for _, d := range dirs {
			size += dirSize(d)
		}
		writeLine(env.Out, "")
		writeLine(env.Out, "The downloaded models are still there: "+formatSize(size)+".")
		writeLine(env.Out, "Remove them too with: dessau uninstall --purge")
		return
	}

	removed, kept := 0, 0
	for _, d := range dirs {
		size := dirSize(d)
		if err := os.RemoveAll(d); err != nil {
			writeLine(env.Err, "warning: "+redact(d, ue.Home)+" could not be removed ("+err.Error()+")")
			*failed = true
			kept += size
			continue
		}
		removed += size
	}

	writeLine(env.Out, "")
	writeLine(env.Out, "removed "+formatSize(removed)+" of downloaded models.")
	if kept > 0 {
		writeLine(env.Out, "Left in place: "+formatSize(kept)+", which this account could not remove.")
	}
}

// dirSize is how much a path holds, following no symbolic link and tolerating
// what it cannot read — a size is a thing to report, never a thing to fail on.
func dirSize(path string) int {
	total := 0
	_ = filepath.WalkDir(path, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += int(info.Size())
		}
		return nil
	})
	return total
}

// formatSize renders a byte count for a person, in the units the rest of the
// product uses.
func formatSize(n int) string {
	const unit = 1024
	if n < unit {
		return strconv.Itoa(n) + " B"
	}
	div, exp := int64(unit), 0
	for m := int64(n) / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	value := float64(n) / float64(div)
	return strconv.FormatFloat(value, 'f', 1, 64) + " " + []string{"KB", "MB", "GB", "TB"}[exp]
}

package registry

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"syscall"
)

// ErrModelCode is the refusal of a model that ships its own code.
//
// Its text is the reason the operator is shown — on the model's card and in
// the error a client entitled to the pool's reasons receives — so it is plain
// words, and it carries no path, which the bound on a recorded load failure
// would otherwise blank (plausibleLoadFailure).
var ErrModelCode = errors.New("this model ships its own code, which Dessau does not run")

// ErrOtherAccount is the refusal of a model whose files belong to an account
// other than the one running Dessau. Plain words and no path, for the same
// reason as ErrModelCode.
var ErrOtherAccount = errors.New("this model's files belong to another account, which Dessau does not load")

// CheckModelCode reports whether the model in dir may be handed to the model
// server: ErrOtherAccount when dir, or the config.json in it, is not owned by
// owner (the uid of the account running Dessau); ErrModelCode when
// config.json names a model_file; and the reader's own error when config.json
// cannot be read as a JSON object (fs.ErrNotExist among them, for a file that
// is not there). nil means none of those.
//
// The pinned model server (mlx-lm 0.31.3, mlx_lm/utils.py load_model) imports
// and executes the file a model's configuration names there, inside the
// model-server child, as this account, at load time:
//
//	if (model_file := config.get("model_file")) is not None:
//	    spec = importlib.util.spec_from_file_location("custom_model", model_path / model_file)
//	    ...
//	    spec.loader.exec_module(arch)
//
// Dessau does not run code a download carries (iss-2610030709283687), so the
// check is that condition, exactly: the top-level key, spelled exactly, any
// value but null — an empty string, a number, an object all refuse, since
// the model server would try each. A key nested in another object, or
// spelled in another case, is not one the model server reads. The decode is
// into a map, so duplicate keys go to the last one, as Python's json module
// takes them, and keys match case-sensitively, as a Python dict does — a
// struct field would match "Model_File" too, and not as the model server
// reads it.
//
// The model server reads config.json again, by path, seconds after this
// check, so the check is worth only as much as the guarantee that nobody
// else can change the file in between. In the shared cache a model's files
// belong to whichever account downloaded them, and the owner of a file can
// rename another over it, as can the owner of the directory it sits in,
// whatever the sticky bit says. So the files are this account's own or they
// are not loaded: the model directory (the link and what it leads to, when it
// is a link) and config.json — the latter's owner read off the very handle
// its bytes came from, so the owner and the content are of one open file.
// Dessau serves from one account; other accounts reach it over the network
// and have no need to load another account's copy of a model.
//
// It goes through readModelConfigInfo, the one decoder of a model's
// configuration, so the file is opened as every manifest is: a link or a
// FIFO is refused, and so is a file over maxManifestJSON. What it cannot read
// it does not vouch for: the caller decides what an unreadable or missing
// file means.
func CheckModelCode(dir string, owner int) error {
	return checkModelCode(dir, func(fi fs.FileInfo) bool {
		uid, ok := fileOwner(fi)
		return ok && int(uid) == owner
	})
}

// checkModelCode is CheckModelCode with the ownership rule as a predicate,
// so a test can hold the directory and config.json to different answers,
// which a test running as one account cannot arrange on disk.
func checkModelCode(dir string, owned func(fs.FileInfo) bool) error {
	lfi, err := os.Lstat(dir)
	if err != nil {
		return err
	}
	if !owned(lfi) {
		return ErrOtherAccount
	}
	if lfi.Mode()&fs.ModeSymlink != 0 {
		fi, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !owned(fi) {
			return ErrOtherAccount
		}
		lfi = fi
	}
	if !lfi.IsDir() {
		return fmt.Errorf("the model's path is not a directory")
	}
	cfg, info, err := readModelConfigInfo(dir)
	if err != nil {
		return err
	}
	if !owned(info) {
		return ErrOtherAccount
	}
	if v, ok := cfg["model_file"]; ok && v != nil {
		return ErrModelCode
	}
	return nil
}

// fileOwner is the uid fi belongs to, when the platform says.
func fileOwner(fi fs.FileInfo) (uint32, bool) {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, false
	}
	return st.Uid, true
}

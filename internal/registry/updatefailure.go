package registry

import "fmt"

// Why a model's last update failed, as a class rather than the error's text:
// the text can carry paths and addresses, and the card shows words of its
// own for each class (iss-2610031320392078). The version being served is
// untouched in every case.
const (
	// UpdateFailedDownload: a file did not download, or did not match the
	// hash the Hub lists for it.
	UpdateFailedDownload = "download"
	// UpdateFailedNoSpace: the models folder had no room for the new
	// version beside the old one.
	UpdateFailedNoSpace = "no_space"
	// UpdateFailedRefused: the new version did not pass the checks made
	// before a model is started.
	UpdateFailedRefused = "refused"
	// UpdateFailedBusy: the model was still answering requests when the
	// swap's wait ran out.
	UpdateFailedBusy = "busy"
	// UpdateFailedNotOffered: the newer version is one Dessau does not run
	// — it ships its own code, or waits for review.
	UpdateFailedNotOffered = "not_offered"
)

func validUpdateFailure(class string) bool {
	switch class {
	case UpdateFailedDownload, UpdateFailedNoSpace, UpdateFailedRefused, UpdateFailedBusy, UpdateFailedNotOffered:
		return true
	}
	return false
}

// SetUpdateFailure records why the model's last update failed. The next
// successful download's record carries none, which is what clears it.
func (r *Registry) SetUpdateFailure(repoID, class string) error {
	if !validUpdateFailure(class) {
		return fmt.Errorf("registry: %q is not an update failure", class)
	}
	r.mu.Lock()
	existing, ok := r.models[key(repoID)]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("registry: %s: %w", repoID, ErrNotFound)
	}
	existing.UpdateFailed = class
	r.models[key(repoID)] = existing
	snapshot := r.listLocked()
	err := r.saveLocked()
	r.mu.Unlock()
	r.broadcast(snapshot)
	return err
}

package capability

import "syscall"

// ramBudgetPercent is the share of physical memory a loaded model may use when
// the operator has set no figure of their own.
//
// Apple Silicon has unified memory: whatever the models take is taken from the
// same pool as the window server and everything else running. 60% keeps a 64 GB
// machine usable while still fitting a 4-bit 70B. This is the one home of that
// share — the process pool resolves its own default through DefaultBudget — so
// the "fits" filter and the pool cannot come to disagree about it.
const ramBudgetPercent = 60

// unmeasuredBudget is what a machine whose memory cannot be read is allowed.
// A budget of nothing would refuse every model, which is worse than a
// conservative guess.
const unmeasuredBudget = 8 << 30

// DefaultBudget is how much memory loaded models may use on a machine of the
// given size, when no budget has been configured.
func DefaultBudget(totalRAM int64) int64 {
	if totalRAM <= 0 {
		return unmeasuredBudget
	}
	return totalRAM * ramBudgetPercent / 100
}

// Assess reports a machine of the given size and budget, measuring the free
// space on the volume that holds modelsDir.
//
// Both figures are parameters rather than ones this package resolves. The
// budget is the operator's setting, and the filter has to hide exactly what the
// process pool would refuse; the machine's size is read once where the app
// holds it, so that every surface of one control panel answers from one
// reading rather than from whatever sysctl says at the moment each is drawn.
// Zero for either — an unmeasurable Mac, a fresh install's stored budget —
// means what it means everywhere else: unknown, and the default share.
func Assess(modelsDir string, totalRAM, budget int64) Machine {
	if totalRAM < 0 {
		totalRAM = 0
	}
	if budget <= 0 {
		budget = DefaultBudget(totalRAM)
	}
	return Machine{
		TotalRAM:  totalRAM,
		RAMBudget: budget,
		FreeDisk:  freeDisk(modelsDir),
	}
}

// freeDisk returns the bytes available to an unprivileged user on the volume
// containing dir, falling back to the root volume if dir does not exist yet.
func freeDisk(dir string) int64 {
	if dir == "" {
		dir = "/"
	}
	n, ok := FreeDisk(dir)
	if !ok {
		n, _ = FreeDisk("/")
	}
	return n
}

// FreeDisk is the bytes available to an unprivileged user on the volume
// holding dir, and whether the system said: the one reader of free disk, for
// the budget above and for a staged update's room check (internal/app), which
// must tell a full disk from one it could not ask about.
func FreeDisk(dir string) (int64, bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(dir, &st); err != nil {
		return 0, false
	}
	return int64(st.Bavail) * int64(st.Bsize), true
}

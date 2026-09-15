package contextprobe

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// The filler is the 2026-09-06 campaign's: ordinary English, varied by a
// marker so the tokenizer sees natural text, and started with a nonce so no
// two prompts share a prefix — the mistake that let a retained prompt cache
// put one model's window a third too high in the earlier record.
const block = "The lighthouse keeper counted the waves as they broke against the seawall, " +
	"marking each seventh one in a small leather notebook. %s " +
	"Morning fog delayed the ferry, and the gulls circled the harbour in slow, " +
	"patient spirals while the fishermen mended their nets on the pier. "

var markers = []string{"Nearby,", "Meanwhile,", "Later,", "Somehow,", "Quietly,", "Often,", "Yesterday,",
	"Today,", "Besides,", "However,", "Beyond,", "Above,", "Below,", "Perhaps,",
	"Certainly,", "Suddenly,", "Gradually,", "Finally,"}

// filler builds about chars characters of text, beginning with its own nonce
// and its own marker order.
func filler(chars int) string {
	nonce := make([]byte, 8)
	_, _ = rand.Read(nonce)
	tag := hex.EncodeToString(nonce)
	// The marker order starts where the nonce says, so two prompts of the
	// same length differ throughout and not only in their first line.
	start := int(nonce[0]) % len(markers)
	var b strings.Builder
	b.Grow(chars + len(block))
	b.WriteString("[" + tag + "] ")
	for i := 0; b.Len() < chars; i++ {
		b.WriteString(strings.Replace(block, "%s", markers[(start+i)%len(markers)], 1))
	}
	return b.String()
}

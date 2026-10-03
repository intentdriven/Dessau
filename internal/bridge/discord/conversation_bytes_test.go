package discord

import (
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"
)

// A turn is bounded in bytes, not runes: a rune is up to four bytes, so a
// bound in runes is a quarter of what it seems in the scripts where it bites
// hardest. It is cut on a rune boundary, so what is kept is still text (the
// 2026-09-20 decision on iss-2609190312188937).
func TestATurnIsBoundedInBytes(t *testing.T) {
	conv := &conversation{}
	long := strings.Repeat("語", maxTurnBytes) // three bytes a rune
	conv.append(turn{Role: roleUser, Content: long})
	got := conv.history()[0].Content
	if len(got) > maxTurnBytes {
		t.Errorf("a turn holds %d bytes, and the bound is %d", len(got), maxTurnBytes)
	}
	if !utf8.ValidString(got) {
		t.Error("the bound cut a rune in half")
	}
	if !strings.HasPrefix(long, got) || len(got) < maxTurnBytes-utf8.UTFMax {
		t.Errorf("the bound kept %d bytes of the head, want as much as fits", len(got))
	}
	if n := len(headBytes(long, maxTurnBytes)); n != len(got) {
		t.Errorf("an incoming message is cut to %d bytes and a stored turn to %d; one bound, not two", n, len(got))
	}
}

// The whole store carries one byte budget across every channel. When it is
// reached the oldest turns go first, whichever channel holds them, and a
// channel that loses every turn keeps its model.
func TestTheStoreHoldsOneByteBudgetAcrossChannels(t *testing.T) {
	c := newConversations()
	full := strings.Repeat("x", maxTurnBytes)
	first := c.get("channel-0")
	first.setModel("the-first-model")
	first.append(turn{Role: roleUser, Content: "the oldest turn"})

	// More turns of the largest size than the budget holds, spread over
	// more channels than there are turns per channel.
	need := maxStoreBytes/maxTurnBytes + 2*maxTurns
	for i := 0; i < need; i++ {
		c.get("channel-" + strconv.Itoa(1+i%(maxChannels-1))).append(turn{Role: roleUser, Content: full})
	}
	if got := c.bytes(); got > maxStoreBytes {
		t.Errorf("the store holds %d bytes of turns, and the budget is %d", got, maxStoreBytes)
	}
	if got := c.counted(); got != c.bytes() {
		t.Errorf("the store's running total is %d and its turns add up to %d", c.bytes(), got)
	}
	for _, tr := range first.history() {
		if tr.Content == "the oldest turn" {
			t.Error("the oldest turn in the store survived the budget")
		}
	}
	if got := c.get("channel-0").modelOf(); got != "the-first-model" {
		t.Errorf("a channel whose turns the budget took lost its model: %q", got)
	}
	// The newest turn is always kept.
	last := c.get("channel-" + strconv.Itoa(1+(need-1)%(maxChannels-1))).history()
	if len(last) == 0 || last[len(last)-1].Content != full {
		t.Error("the newest turn did not survive the budget")
	}
}

// The counts come down with the byte bounds, and the store's ceiling is its
// budget rather than their product.
func TestTheStoresCeilingIsSmall(t *testing.T) {
	if maxChannels*maxTurns >= 256*64 {
		t.Errorf("%d channels of %d turns is no lower than the 256 of 64 the decision lowered", maxChannels, maxTurns)
	}
	if maxStoreBytes > 8<<20 {
		t.Errorf("the store's byte budget is %d, want at most 8 MiB", maxStoreBytes)
	}
}

// /reset and a channel going from the store both give their bytes back.
func TestResetAndEvictionReturnTheirBytes(t *testing.T) {
	c := newConversations()
	conv := c.get("channel-0")
	conv.append(turn{Role: roleUser, Content: "hello"})
	conv.reset()
	if got := c.bytes(); got != 0 {
		t.Errorf("after /reset the store still counts %d bytes", got)
	}
	c.get("channel-0").append(turn{Role: roleUser, Content: "hello"})
	for i := 1; i <= maxChannels; i++ {
		c.get("channel-" + strconv.Itoa(i))
	}
	if got := c.bytes(); got != 0 {
		t.Errorf("after the channel went the store still counts %d bytes", got)
	}
	// A conversation that left the store, still held by an answer in flight,
	// counts nothing towards it.
	conv.append(turn{Role: roleAssistant, Content: "late"})
	if got := c.bytes(); got != 0 {
		t.Errorf("a turn appended to a conversation the store let go was counted: %d", got)
	}
}

// A conversation held to its turn count gives back the bytes of the turns
// the count drops: otherwise the store's total runs ahead of what it holds
// and the budget empties every channel on each append.
func TestTheTurnCountGivesItsBytesBack(t *testing.T) {
	c := newConversations()
	conv := c.get("channel-0")
	for i := range maxTurns * 3 {
		conv.append(turn{Role: roleUser, Content: strconv.Itoa(i)})
	}
	if got, want := c.bytes(), c.counted(); got != want {
		t.Errorf("after the turn count dropped turns the store counts %d bytes and holds %d", got, want)
	}
}

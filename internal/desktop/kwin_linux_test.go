package desktop

import "testing"

func TestSnapshotRejectsSpoofedAndStaleReplies(t *testing.T) {
	ch := make(chan Window, 1)
	k := &KWin{pending: map[string]pending{"nonce": {owner: ":1.42", result: ch}}}
	if k.Snapshot(":1.99", "nonce", "id", "wow.exe", "World of Warcraft") == nil {
		t.Fatal("accepted wrong D-Bus sender")
	}
	if k.Snapshot(":1.42", "stale", "id", "wow.exe", "World of Warcraft") == nil {
		t.Fatal("accepted stale reply")
	}
	if err := k.Snapshot(":1.42", "nonce", "id", "wow.exe", "World of Warcraft"); err != nil {
		t.Fatal(err)
	}
	select {
	case w := <-ch:
		if w.ID != "id" {
			t.Fatal(w)
		}
	default:
		t.Fatal("missing KWin reply")
	}
}

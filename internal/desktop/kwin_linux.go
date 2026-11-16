// Package desktop implements the Plasma 6 session integration. No shell is used.
package desktop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/godbus/dbus/v5"
)

const bridgeInterface = "org.afishk.Companion"
const bridgePath dbus.ObjectPath = "/org/afishk/Companion"

type pending struct {
	owner  string
	result chan Window
}
type KWin struct {
	conn    *dbus.Conn
	queryMu sync.Mutex
	mu      sync.Mutex
	pending map[string]pending
}

func OpenKWin() (*KWin, error) {
	c, err := dbus.ConnectSessionBus()
	if err != nil {
		return nil, err
	}
	k := &KWin{conn: c, pending: make(map[string]pending)}
	if err = c.Export(k, bridgePath, bridgeInterface); err != nil {
		_ = c.Close()
		return nil, err
	}
	return k, nil
}
func (k *KWin) Close() { _ = k.conn.Close() }

// Snapshot is called only by a freshly loaded KWin script. The sender must be
// KWin's unique bus owner; nonces prevent stale replies from satisfying a query.
func (k *KWin) Snapshot(sender dbus.Sender, nonce, id, class, title string) *dbus.Error {
	k.mu.Lock()
	defer k.mu.Unlock()
	p, ok := k.pending[nonce]
	if !ok || string(sender) != p.owner {
		return dbus.MakeFailedError(errors.New("unexpected KWin reply"))
	}
	select {
	case p.result <- Window{id, class, title}:
	default:
	}
	return nil
}

func (k *KWin) Active(ctx context.Context) (Window, error) {
	k.queryMu.Lock()
	defer k.queryMu.Unlock()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	var locked bool
	if err := k.conn.Object("org.freedesktop.ScreenSaver", "/ScreenSaver").CallWithContext(ctx, "org.freedesktop.ScreenSaver.GetActive", 0).Store(&locked); err != nil {
		return Window{}, fmt.Errorf("cannot verify unlocked desktop: %w", err)
	}
	if locked {
		return Window{}, errors.New("desktop is locked")
	}
	var owner string
	if err := k.conn.BusObject().CallWithContext(ctx, "org.freedesktop.DBus.GetNameOwner", 0, "org.kde.KWin").Store(&owner); err != nil {
		return Window{}, err
	}
	nonceBytes := make([]byte, 16)
	if _, err := rand.Read(nonceBytes); err != nil {
		return Window{}, err
	}
	nonce := hex.EncodeToString(nonceBytes)
	ch := make(chan Window, 1)
	k.mu.Lock()
	k.pending[nonce] = pending{owner, ch}
	k.mu.Unlock()
	defer func() { k.mu.Lock(); delete(k.pending, nonce); k.mu.Unlock() }()
	// JSON quoting safely embeds the unique D-Bus connection name and nonce.
	busName, _ := json.Marshal(k.conn.Names()[0])
	n, _ := json.Marshal(nonce)
	script := fmt.Sprintf(`var w = workspace.activeWindow;
callDBus(%s, "/org/afishk/Companion", "org.afishk.Companion", "Snapshot", %s,
 w ? String(w.internalId) : "", w ? String(w.resourceClass) : "", w ? String(w.caption) : "");`, busName, n)
	f, err := os.CreateTemp("", "afishk-*.js")
	if err != nil {
		return Window{}, err
	}
	defer func() { _ = os.Remove(f.Name()) }()
	if _, err = f.WriteString(script); err != nil {
		_ = f.Close()
		return Window{}, err
	}
	if err = f.Close(); err != nil {
		return Window{}, err
	}
	scripting := k.conn.Object(owner, "/Scripting")
	var id int32
	if err = scripting.CallWithContext(ctx, "org.kde.kwin.Scripting.loadScript", 0, f.Name(), "afishk-"+nonce).Store(&id); err != nil {
		return Window{}, err
	}
	if id < 0 {
		return Window{}, errors.New("KWin refused focus script")
	}
	defer func() {
		cleanup, c := context.WithTimeout(context.Background(), time.Second)
		defer c()
		scripting.CallWithContext(cleanup, "org.kde.kwin.Scripting.unloadScript", 0, "afishk-"+nonce)
	}()
	obj := k.conn.Object(owner, dbus.ObjectPath(fmt.Sprintf("/Scripting/Script%d", id)))
	if err = obj.CallWithContext(ctx, "org.kde.kwin.Script.run", 0).Err; err != nil {
		return Window{}, err
	}
	select {
	case w := <-ch:
		if w.ID == "" {
			return w, errors.New("no active window")
		}
		return w, nil
	case <-ctx.Done():
		return Window{}, fmt.Errorf("KWin focus query: %w", ctx.Err())
	}
}

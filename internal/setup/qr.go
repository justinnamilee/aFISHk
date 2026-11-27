package setup

import (
	"fmt"
	"io"
	"strings"

	"rsc.io/qr"
)

func printQR(out io.Writer, origin string) error {
	code, err := qr.Encode(origin, qr.M)
	if err != nil {
		_, err = fmt.Fprintln(out, "The URL is too long for a QR code. Open the address shown above.")
		return err
	}
	const margin = 4
	const reset = "\x1b[0m"
	// A half block packs two QR rows into one terminal row. Set both colors
	// explicitly so the code scans with either light or dark terminal themes.
	colors := [4]string{"\x1b[97;107m", "\x1b[30;107m", "\x1b[97;40m", "\x1b[30;40m"}
	var b strings.Builder
	b.WriteString("Scan to open aFISHk on your phone:\n")
	for y := -margin; y < code.Size+margin; y += 2 {
		previous := -1
		for x := -margin; x < code.Size+margin; x++ {
			pair := 0
			if code.Black(x, y) {
				pair |= 1
			}
			if code.Black(x, y+1) {
				pair |= 2
			}
			if pair != previous {
				b.WriteString(colors[pair])
				previous = pair
			}
			b.WriteRune('▀')
		}
		b.WriteString(reset + "\n")
	}
	_, err = io.WriteString(out, b.String())
	return err
}

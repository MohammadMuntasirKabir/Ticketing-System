// Package seat encodes and decodes aircraft seat labels (e.g. "12A")
// into stable numeric seat ids. The encoding is deterministic so any
// layer can map a label to an id without a database round-trip, and the
// seat-map builder in the repository generates ids with the same scheme.
package seat

import (
	"fmt"
	"strings"
)

const colBits = 100

// EncodeID packs a seat's 1-based row and column letter into a stable,
// unique id. Columns are limited to A..Z.
func EncodeID(row int, col string) int64 {
	idx := columnIndex(col)
	if row <= 0 || idx == 0 {
		return 0
	}
	return int64(row)*colBits + int64(idx)
}

// DecodeLabel renders the "12A"-style label for a seat id, or "" when the
// id is not representable.
func DecodeLabel(id int64) string {
	if id <= 0 {
		return ""
	}
	row := id / colBits
	idx := id % colBits
	if row <= 0 || idx <= 0 || idx > 26 {
		return ""
	}
	return fmt.Sprintf("%d%c", row, 'A'+byte(idx)-1)
}

// ParseLabel splits "12A" into its numeric row and column letter.
func ParseLabel(label string) (row int, col string, ok bool) {
	label = strings.ToUpper(strings.TrimSpace(label))
	i := 0
	for i < len(label) && label[i] >= '0' && label[i] <= '9' {
		i++
	}
	if i == 0 || i >= len(label) {
		return 0, "", false
	}
	for j := 0; j < i; j++ {
		row = row*10 + int(label[j]-'0')
	}
	col = label[i:]
	if row <= 0 || len(col) != 1 || col[0] < 'A' || col[0] > 'Z' {
		return 0, "", false
	}
	return row, col, true
}

func columnIndex(col string) int {
	c := strings.ToUpper(strings.TrimSpace(col))
	if len(c) != 1 || c[0] < 'A' || c[0] > 'Z' {
		return 0
	}
	return int(c[0]-'A') + 1
}

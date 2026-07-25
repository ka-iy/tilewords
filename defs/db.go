package defs

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"
)

// DB is a set of headword definitions with the inflection edges needed to resolve
// a played word to one of them. It is immutable after construction and safe for
// concurrent use by Lookup.
//
// The headwords, glosses and inflected forms are held in flat parallel arrays rather
// than in maps — the same compressed-sparse-row shape the dictionary GADDAGs use. A
// map of pointers costs a bucket slot, a string header and a per-entry allocation for
// every headword, which for an asset this size dominates the process's heap; the flat
// form stores the text once in a byte blob and reaches it through offset arrays, with
// no per-entry allocation. Lookups binary-search the (sorted) blobs instead of hashing,
// and materialise an Entry only for the word actually found.
type DB struct {
	// headBlob holds every headword concatenated in ascending order, no separators.
	headBlob []byte
	// headOff delimits headword i as headBlob[headOff[i]:headOff[i+1]]. Length is Len()+1,
	// so the final element is the sentinel len(headBlob).
	headOff []uint32
	// senseOff delimits headword i's senses as the index range
	// [senseOff[i], senseOff[i+1]) into glossOff and sensePOS. Length is Len()+1.
	senseOff []uint32
	// glossBlob holds every gloss concatenated, in headword order and, within a headword,
	// in sense order.
	glossBlob []byte
	// glossOff delimits sense j as glossBlob[glossOff[j]:glossOff[j+1]]. Length is the
	// total sense count plus a sentinel.
	glossOff []uint32
	// sensePOS[j] is sense j's part of speech, as an index into posTable.
	sensePOS []uint32
	// posTable holds the distinct part-of-speech strings. There are only a handful, so
	// interning them keeps one copy instead of one per sense.
	posTable []string
	// formBlob holds every inflected form concatenated in ascending order, no separators.
	formBlob []byte
	// formOff delimits form k as formBlob[formOff[k]:formOff[k+1]]. Length is
	// FormCount()+1.
	formOff []uint32
	// formLemma[k] is the index of the headword that form k resolves to.
	formLemma []uint32
}

// The on-disk asset is a gzip-compressed byte stream, written and read field by field in
// the order below. It is deliberately not a gob (or any other reflective) encoding: those
// buffer a whole message before handing it over, which for an asset this size doubles peak
// memory at load — the cost that matters on a phone, where the startup spike is what gets
// a process killed. Streaming means the only large allocations are the structures the DB
// keeps, each sized exactly from a count read before it.
//
//	magic      assetMagic
//	counts     headword, sense and form counts; part-of-speech count; the three blob sizes
//	posTable   each part of speech as a length-prefixed string
//	headBlob   the headword bytes, then one length per headword
//	senses     one sense count per headword
//	glossBlob  the gloss bytes, then one length per sense
//	sensePOS   one part-of-speech index per sense
//	formBlob   the form bytes, then one length per form, then one lemma index per form
//
// Every count and length is an unsigned varint, which is what keeps the asset small:
// lengths are small numbers costing a byte or two, whereas the absolute offsets the DB
// uses at runtime are large and compress poorly. Offsets are rebuilt by prefix sum while
// reading, so they cannot drift from the blobs they index.
const (
	// assetMagic identifies the asset and its layout. Change it whenever the layout
	// changes, so a stale asset is refused with a clear message instead of being
	// misparsed into nonsense.
	assetMagic = "TWDEFS\x01\n"

	// maxBlobLen bounds a declared blob size. A corrupt length must not be handed
	// straight to make(), which would abort the process on an absurd allocation; no
	// legitimate asset comes close to this.
	maxBlobLen = 1 << 31
)

// NewDB builds a DB from a headword map and an inflection-edge map. Neither map is
// retained: both are read into the flat representation described on DB.
//
// An edge whose lemma is not itself a headword is dropped, because such an edge can
// never resolve to a definition — Lookup requires the lemma's entry to report a match.
func NewDB(entries map[string]*Entry, formOf map[string]string) *DB {
	words := make([]string, 0, len(entries))
	for w, e := range entries {
		if e == nil {
			continue
		}
		words = append(words, w)
	}
	sort.Strings(words)

	db := &DB{
		headOff:  make([]uint32, 1, len(words)+1),
		senseOff: make([]uint32, 1, len(words)+1),
		glossOff: []uint32{0},
	}

	index := make(map[string]uint32, len(words))
	posIdx := make(map[string]uint32)

	for i, w := range words {
		index[w] = uint32(i)
		db.headBlob = append(db.headBlob, w...)
		db.headOff = append(db.headOff, uint32(len(db.headBlob)))

		for _, s := range entries[w].Senses {
			pi, ok := posIdx[s.POS]
			if !ok {
				pi = uint32(len(db.posTable))
				posIdx[s.POS] = pi
				db.posTable = append(db.posTable, s.POS)
			}
			db.sensePOS = append(db.sensePOS, pi)
			db.glossBlob = append(db.glossBlob, s.Gloss...)
			db.glossOff = append(db.glossOff, uint32(len(db.glossBlob)))
		}
		db.senseOff = append(db.senseOff, uint32(len(db.glossOff)-1))
	}

	forms := make([]string, 0, len(formOf))
	for f, lemma := range formOf {
		if _, ok := index[lemma]; !ok {
			continue
		}
		forms = append(forms, f)
	}
	sort.Strings(forms)

	db.formOff = make([]uint32, 1, len(forms)+1)
	db.formLemma = make([]uint32, 0, len(forms))
	for _, f := range forms {
		db.formBlob = append(db.formBlob, f...)
		db.formOff = append(db.formOff, uint32(len(db.formBlob)))
		db.formLemma = append(db.formLemma, index[formOf[f]])
	}

	return db
}

// Len returns the number of headwords in the DB.
func (db *DB) Len() int { return len(db.headOff) - 1 }

// FormCount returns the number of inflection edges in the DB.
func (db *DB) FormCount() int { return len(db.formLemma) }

// headAt returns headword i's bytes, aliasing headBlob rather than copying.
func (db *DB) headAt(i int) []byte {
	return db.headBlob[db.headOff[i]:db.headOff[i+1]]
}

// formAt returns inflected form k's bytes, aliasing formBlob rather than copying.
func (db *DB) formAt(k int) []byte {
	return db.formBlob[db.formOff[k]:db.formOff[k+1]]
}

// findHead returns the index of headword word, and whether it is present.
func (db *DB) findHead(word string) (int, bool) {
	q := []byte(word)
	n := db.Len()
	i := sort.Search(n, func(i int) bool { return bytes.Compare(db.headAt(i), q) >= 0 })
	if i < n && bytes.Equal(db.headAt(i), q) {
		return i, true
	}
	return 0, false
}

// findForm returns the index of inflected form word, and whether it is present.
func (db *DB) findForm(word string) (int, bool) {
	q := []byte(word)
	n := db.FormCount()
	k := sort.Search(n, func(k int) bool { return bytes.Compare(db.formAt(k), q) >= 0 })
	if k < n && bytes.Equal(db.formAt(k), q) {
		return k, true
	}
	return 0, false
}

// entryAt materialises headword i's Entry. The strings it returns are copies, so a
// caller cannot reach into the DB's blobs.
func (db *DB) entryAt(i int) *Entry {
	lo, hi := db.senseOff[i], db.senseOff[i+1]
	e := &Entry{Word: string(db.headAt(i))}
	if hi > lo {
		e.Senses = make([]Sense, 0, hi-lo)
	}
	for j := lo; j < hi; j++ {
		e.Senses = append(e.Senses, Sense{
			POS:   db.posTable[db.sensePOS[j]],
			Gloss: string(db.glossBlob[db.glossOff[j]:db.glossOff[j+1]]),
		})
	}
	return e
}

// entriesMap rebuilds the headword map. It is used where a whole-DB map is genuinely
// wanted — merging in a supplement, and encoding — not on the lookup path.
func (db *DB) entriesMap() map[string]*Entry {
	m := make(map[string]*Entry, db.Len())
	for i := 0; i < db.Len(); i++ {
		m[string(db.headAt(i))] = db.entryAt(i)
	}
	return m
}

// formsMap rebuilds the inflection-edge map, for the same reasons as entriesMap.
func (db *DB) formsMap() map[string]string {
	m := make(map[string]string, db.FormCount())
	for k := 0; k < db.FormCount(); k++ {
		m[string(db.formAt(k))] = string(db.headAt(int(db.formLemma[k])))
	}
	return m
}

// WithSupplement returns a new DB that layers supplemental headword entries and
// inflection edges on top of db without mutating db. Existing keys take
// precedence: a supplemental entry or edge whose key is already present in db is
// dropped, so the authoritative (primary-source) definitions and edges are never
// overwritten by a lower-priority source. Both argument maps are read, not
// retained. It is used to fold definitions from secondary public-domain
// dictionaries into the DB for words the primary source does not cover.
func (db *DB) WithSupplement(entries map[string]*Entry, forms map[string]string) *DB {
	mergedEntries := db.entriesMap()
	for k, v := range entries {
		if _, exists := mergedEntries[k]; !exists {
			mergedEntries[k] = v
		}
	}

	mergedForms := db.formsMap()
	for k, v := range forms {
		if _, exists := mergedForms[k]; !exists {
			mergedForms[k] = v
		}
	}

	return NewDB(mergedEntries, mergedForms)
}

// FormLemma returns the lemma an inflected form maps to and whether such an edge
// exists. It reports the raw edge regardless of whether word is also a headword,
// so callers can detect a word that is both a headword and an inflection.
func (db *DB) FormLemma(word string) (string, bool) {
	k, ok := db.findForm(strings.ToLower(strings.TrimSpace(word)))
	if !ok {
		return "", false
	}
	return string(db.headAt(int(db.formLemma[k]))), true
}

// lemmaOf returns the headword index an inflected form resolves to.
func (db *DB) lemmaOf(word string) (int, bool) {
	k, ok := db.findForm(word)
	if !ok {
		return 0, false
	}
	return int(db.formLemma[k]), true
}

// Lookup resolves word to a definition, trying each resolution layer in order of
// reliability (exact, form-of, stem, fuzzy). The match is case-insensitive.
// The second return is false when no layer produced a match.
func (db *DB) Lookup(word string) (Result, bool) {
	lw := strings.ToLower(strings.TrimSpace(word))
	if lw == "" {
		return Result{}, false
	}

	if i, ok := db.findHead(lw); ok {
		res := Result{Entry: db.entryAt(i), Headword: lw, Kind: MatchExact}
		// A word can be both a headword and an inflected form of another lemma
		// (e.g. "mice" is a rare verb yet chiefly the plural of "mouse"). Surface
		// the lemma's senses too rather than hiding the common reading behind the
		// homograph. The word's own senses stay primary, so this never demotes a
		// common homograph (e.g. "rose", "found") to its inflected reading.
		if li, ok := db.lemmaOf(lw); ok && li != i {
			res.AlsoForm = db.entryAt(li)
			res.AlsoFormWord = string(db.headAt(li))
		}
		return res, true
	}

	if li, ok := db.lemmaOf(lw); ok {
		return Result{Entry: db.entryAt(li), Headword: string(db.headAt(li)), Kind: MatchFormOf}, true
	}

	for _, c := range candidateStems(lw) {
		if r, ok := db.resolveHeadword(c, MatchStem); ok {
			return r, true
		}
	}

	for _, c := range variantCandidates(lw) {
		if r, ok := db.resolveHeadword(c, MatchFuzzy); ok {
			return r, true
		}
	}

	return Result{}, false
}

// resolveHeadword returns the definition reached by candidate cand — either as a
// headword directly or via an inflection edge to a lemma — tagged with kind.
func (db *DB) resolveHeadword(cand string, kind MatchKind) (Result, bool) {
	if i, ok := db.findHead(cand); ok {
		return Result{Entry: db.entryAt(i), Headword: cand, Kind: kind}, true
	}
	if li, ok := db.lemmaOf(cand); ok {
		return Result{Entry: db.entryAt(li), Headword: string(db.headAt(li)), Kind: kind}, true
	}
	return Result{}, false
}

// writeUvarint writes v to w through scratch, which must be at least
// binary.MaxVarintLen64 long.
func writeUvarint(w io.Writer, scratch []byte, v uint64) error {
	n := binary.PutUvarint(scratch, v)
	_, err := w.Write(scratch[:n])
	return err
}

// writeLengths writes one varint per item, derived from an offset array's successive
// differences — the on-disk counterpart of readOffsets.
func writeLengths(w io.Writer, scratch []byte, off []uint32) error {
	for i := 0; i+1 < len(off); i++ {
		if err := writeUvarint(w, scratch, uint64(off[i+1]-off[i])); err != nil {
			return err
		}
	}
	return nil
}

// writeUint32s writes one varint per element.
func writeUint32s(w io.Writer, scratch []byte, xs []uint32) error {
	for _, x := range xs {
		if err := writeUvarint(w, scratch, uint64(x)); err != nil {
			return err
		}
	}
	return nil
}

// Encode writes the DB to w in the gzip-compressed layout documented above. The bytes are
// deterministic: the flat arrays have a fixed order, so re-encoding an unchanged DB
// reproduces the same asset (see TestEncodeIsDeterministic).
func (db *DB) Encode(w io.Writer) error {
	gz := gzip.NewWriter(w)
	bw := bufio.NewWriterSize(gz, 1<<16)
	scratch := make([]byte, binary.MaxVarintLen64)

	err := func() error {
		if _, err := bw.WriteString(assetMagic); err != nil {
			return err
		}
		for _, v := range []uint64{
			uint64(db.Len()), uint64(len(db.sensePOS)), uint64(db.FormCount()),
			uint64(len(db.posTable)),
			uint64(len(db.headBlob)), uint64(len(db.glossBlob)), uint64(len(db.formBlob)),
		} {
			if err := writeUvarint(bw, scratch, v); err != nil {
				return err
			}
		}
		for _, pos := range db.posTable {
			if err := writeUvarint(bw, scratch, uint64(len(pos))); err != nil {
				return err
			}
			if _, err := bw.WriteString(pos); err != nil {
				return err
			}
		}
		if _, err := bw.Write(db.headBlob); err != nil {
			return err
		}
		if err := writeLengths(bw, scratch, db.headOff); err != nil {
			return err
		}
		if err := writeLengths(bw, scratch, db.senseOff); err != nil {
			return err
		}
		if _, err := bw.Write(db.glossBlob); err != nil {
			return err
		}
		if err := writeLengths(bw, scratch, db.glossOff); err != nil {
			return err
		}
		if err := writeUint32s(bw, scratch, db.sensePOS); err != nil {
			return err
		}
		if _, err := bw.Write(db.formBlob); err != nil {
			return err
		}
		if err := writeLengths(bw, scratch, db.formOff); err != nil {
			return err
		}
		return writeUint32s(bw, scratch, db.formLemma)
	}()
	if err != nil {
		gz.Close()
		return fmt.Errorf("defs.Encode: %w", err)
	}

	if err := bw.Flush(); err != nil {
		gz.Close()
		return fmt.Errorf("defs.Encode: %w", err)
	}
	if err := gz.Close(); err != nil {
		return fmt.Errorf("defs.Encode: flush gzip: %w", err)
	}
	return nil
}

// readCount reads a varint that will be used to size an allocation, rejecting anything
// beyond limit so a corrupt asset cannot demand an absurd amount of memory.
func readCount(br io.ByteReader, limit uint64, what string) (int, error) {
	v, err := binary.ReadUvarint(br)
	if err != nil {
		return 0, fmt.Errorf("read %s: %w", what, err)
	}
	if v > limit {
		return 0, fmt.Errorf("%s is %d, beyond the %d maximum", what, v, limit)
	}
	return int(v), nil
}

// readBlob reads exactly n bytes into a slice sized for them.
func readBlob(r io.Reader, n int, what string) ([]byte, error) {
	if n == 0 {
		return nil, nil
	}
	b := make([]byte, n)
	if _, err := io.ReadFull(r, b); err != nil {
		return nil, fmt.Errorf("read %s: %w", what, err)
	}
	return b, nil
}

// readOffsets reads n item lengths and accumulates them into an offset array with a
// trailing sentinel. The lengths must sum to exactly total — the check that stops a
// malformed asset from yielding out-of-range slice indexes later.
func readOffsets(br io.ByteReader, n, total int, what string) ([]uint32, error) {
	off := make([]uint32, n+1)
	var sum uint64
	for i := 0; i < n; i++ {
		v, err := binary.ReadUvarint(br)
		if err != nil {
			return nil, fmt.Errorf("read %s length %d: %w", what, i, err)
		}
		sum += v
		if sum > uint64(total) {
			return nil, fmt.Errorf("%s lengths overrun the blob at index %d", what, i)
		}
		off[i+1] = uint32(sum)
	}
	if sum != uint64(total) {
		return nil, fmt.Errorf("%s lengths cover %d of %d blob bytes", what, sum, total)
	}
	return off, nil
}

// readIndexes reads n varints, each of which must address one of limit items.
func readIndexes(br io.ByteReader, n, limit int, what string) ([]uint32, error) {
	if n == 0 {
		return nil, nil
	}
	xs := make([]uint32, n)
	for i := range xs {
		v, err := binary.ReadUvarint(br)
		if err != nil {
			return nil, fmt.Errorf("read %s %d: %w", what, i, err)
		}
		if v >= uint64(limit) {
			return nil, fmt.Errorf("%s %d addresses item %d of %d", what, i, v, limit)
		}
		xs[i] = uint32(v)
	}
	return xs, nil
}

// Decode reads a DB previously written by Encode. Counts, lengths and indexes are all
// validated as they are read, so a corrupt or truncated asset is reported as an error
// rather than panicking on first use — and nothing larger than the DB's own structures is
// allocated on the way, which is what keeps the load spike down.
func Decode(r io.Reader) (*DB, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}
	defer gz.Close()
	br := bufio.NewReaderSize(gz, 1<<16)

	magic := make([]byte, len(assetMagic))
	if _, err := io.ReadFull(br, magic); err != nil {
		return nil, fmt.Errorf("defs.Decode: read header: %w", err)
	}
	if string(magic) != assetMagic {
		return nil, fmt.Errorf("defs.Decode: not a definitions asset of this version; rebuild it with 'make defs'")
	}

	db := &DB{}
	var nHead, nSense, nForm, nPOS, headLen, glossLen, formLen int
	for _, f := range []struct {
		dst   *int
		limit uint64
		what  string
	}{
		{&nHead, maxBlobLen, "headword count"},
		{&nSense, maxBlobLen, "sense count"},
		{&nForm, maxBlobLen, "form count"},
		{&nPOS, maxBlobLen, "part-of-speech count"},
		{&headLen, maxBlobLen, "headword blob size"},
		{&glossLen, maxBlobLen, "gloss blob size"},
		{&formLen, maxBlobLen, "form blob size"},
	} {
		if *f.dst, err = readCount(br, f.limit, f.what); err != nil {
			return nil, fmt.Errorf("defs.Decode: %w", err)
		}
	}

	if nPOS > 0 {
		db.posTable = make([]string, nPOS)
		for i := range db.posTable {
			n, err := readCount(br, maxBlobLen, "part-of-speech length")
			if err != nil {
				return nil, fmt.Errorf("defs.Decode: %w", err)
			}
			b, err := readBlob(br, n, "part of speech")
			if err != nil {
				return nil, fmt.Errorf("defs.Decode: %w", err)
			}
			db.posTable[i] = string(b)
		}
	}

	if db.headBlob, err = readBlob(br, headLen, "headword blob"); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}
	if db.headOff, err = readOffsets(br, nHead, headLen, "headword"); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}
	// senseOff counts senses per headword, so its running sum indexes the sense arrays
	// rather than a byte blob; it must land exactly on the declared sense count.
	if db.senseOff, err = readOffsets(br, nHead, nSense, "sense"); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}

	if db.glossBlob, err = readBlob(br, glossLen, "gloss blob"); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}
	if db.glossOff, err = readOffsets(br, nSense, glossLen, "gloss"); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}
	if db.sensePOS, err = readIndexes(br, nSense, nPOS, "part-of-speech index"); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}

	if db.formBlob, err = readBlob(br, formLen, "form blob"); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}
	if db.formOff, err = readOffsets(br, nForm, formLen, "form"); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}
	if db.formLemma, err = readIndexes(br, nForm, nHead, "form lemma"); err != nil {
		return nil, fmt.Errorf("defs.Decode: %w", err)
	}

	return db, nil
}

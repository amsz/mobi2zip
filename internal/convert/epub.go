package convert

import (
	"archive/zip"
	"encoding/xml"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
)

// imageExts are the pictures a document's pages hold.
var imageExts = map[string]bool{
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
	".webp": true, ".bmp": true, ".avif": true,
}

// openEPUB opens the zip of an EPUB and returns its pictures in the order
// the book shows them.
func openEPUB(file string) (*zip.ReadCloser, []*zip.File, error) {
	r, err := zip.OpenReader(file)
	if err != nil {
		return nil, nil, err
	}
	files, err := epubPictures(r.File)
	if err != nil {
		r.Close()
		return nil, nil, err
	}
	return r, files, nil
}

// epubPictures reads the OPF of an EPUB and walks its spine, collecting the
// pictures each page shows, then any the spine did not reference.
func epubPictures(all []*zip.File) ([]*zip.File, error) {
	byName := make(map[string]*zip.File, len(all))
	for _, f := range all {
		byName[f.Name] = f
	}

	opf := ""
	if data, ok := readEntry(byName, "META-INF/container.xml"); ok {
		var c struct {
			Rootfiles []struct {
				FullPath string `xml:"full-path,attr"`
			} `xml:"rootfiles>rootfile"`
		}
		if xml.Unmarshal(data, &c) == nil && len(c.Rootfiles) > 0 {
			opf = strings.TrimPrefix(c.Rootfiles[0].FullPath, "/")
		}
	}
	if opf == "" {
		for _, f := range all {
			if strings.HasSuffix(strings.ToLower(f.Name), ".opf") {
				opf = f.Name
				break
			}
		}
	}
	if opf == "" {
		return nil, errNoPackage
	}

	data, ok := readEntry(byName, opf)
	if !ok {
		return nil, errNoPackage
	}
	var pkg struct {
		Manifest []struct {
			ID   string `xml:"id,attr"`
			Href string `xml:"href,attr"`
		} `xml:"manifest>item"`
		Spine []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"spine>itemref"`
	}
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	base := path.Dir(opf)
	href := make(map[string]string, len(pkg.Manifest))
	for _, it := range pkg.Manifest {
		href[it.ID] = it.Href
	}

	var ordered []*zip.File
	seen := make(map[string]bool)
	for _, ref := range pkg.Spine {
		page := resolve(base, href[ref.IDRef])
		pageFile, ok := byName[page]
		if !ok {
			continue
		}
		body, err := readAll(pageFile)
		if err != nil {
			continue
		}
		for _, m := range imgRef.FindAllStringSubmatch(string(body), -1) {
			name := resolve(path.Dir(page), m[1])
			f, ok := byName[name]
			if !ok || !isImage(name) || seen[name] {
				continue
			}
			seen[name] = true
			ordered = append(ordered, f)
		}
	}

	// Pictures the pages do not show, such as a cover the spine skips.
	var rest []*zip.File
	for _, f := range all {
		if isImage(f.Name) && !seen[f.Name] {
			rest = append(rest, f)
		}
	}
	sort.SliceStable(rest, func(i, j int) bool { return rest[i].Name < rest[j].Name })
	return append(ordered, rest...), nil
}

// imgRef finds the pictures a page shows, in <img src> and <image
// xlink:href>.
var imgRef = regexp.MustCompile(`(?i)(?:src|xlink:href)\s*=\s*["']([^"']+)["']`)

// resolve joins a relative href to the directory it is in, dropping any
// fragment and query.
func resolve(dir, ref string) string {
	ref = strings.TrimSpace(ref)
	if i := strings.IndexAny(ref, "?#"); i >= 0 {
		ref = ref[:i]
	}
	if ref == "" || strings.Contains(ref, "://") {
		return ""
	}
	ref = strings.TrimPrefix(ref, "/")
	if dir == "." {
		return path.Clean(ref)
	}
	return path.Clean(path.Join(dir, ref))
}

// isImage reports whether name is a picture.
func isImage(name string) bool {
	return imageExts[strings.ToLower(path.Ext(name))]
}

// readEntry reads a file of the zip by its name.
func readEntry(byName map[string]*zip.File, name string) ([]byte, bool) {
	f, ok := byName[name]
	if !ok {
		return nil, false
	}
	data, err := readAll(f)
	if err != nil {
		return nil, false
	}
	return data, true
}

// readAll reads a file of the zip.
func readAll(f *zip.File) ([]byte, error) {
	r, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}

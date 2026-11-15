package domain

// maxFileSize bounds a document's file: a PDF of a long invoice is a few
// hundred KiB.
const maxFileSize = 10 << 20

// FileKind is a file of an electronic document.
type FileKind string

// The files of a document.
const (
	// FileXML is the signed XML (the attached document the DIAN validated).
	FileXML FileKind = "xml"
	// FilePDF is its graphic representation, with the QR.
	FilePDF FileKind = "pdf"
)

// ParseFileKind returns the FileKind of s, or ErrInvalidFileKind.
func ParseFileKind(s string) (FileKind, error) {
	switch k := FileKind(s); k {
	case FileXML, FilePDF:
		return k, nil
	}
	return "", ErrInvalidFileKind
}

// ContentType is the media type of the file's kind.
func (k FileKind) ContentType() string {
	switch k {
	case FileXML:
		return "application/xml"
	case FilePDF:
		return "application/pdf"
	}
	return ""
}

// File is a file of a document, as the provider returned it.
type File struct {
	Kind    FileKind
	Content []byte
}

// NewFile validates a file: a known kind (ErrInvalidFileKind), with content
// (ErrEmptyFile) of at most 10 MiB (ErrFileTooLarge).
func NewFile(kind string, content []byte) (File, error) {
	k, err := ParseFileKind(kind)
	if err != nil {
		return File{}, err
	}
	if len(content) == 0 {
		return File{}, ErrEmptyFile
	}
	if len(content) > maxFileSize {
		return File{}, ErrFileTooLarge
	}
	return File{Kind: k, Content: content}, nil
}

package formatter

// EOLKind represents the line ending style.
type EOLKind string

const (
	EOLLF   EOLKind = "\n"
	EOLCRLF EOLKind = "\r\n"
)

// LineKind classifies what kind of content a line has.
type LineKind int

const (
	LineBlank           LineKind = iota
	LineCommentOnly              // #, //, /* ... */
	LineAttributeHeader          // key = value (start line)
	LineBlockOpen                // label { or label label {
	LineBlockClose               // }
	LineOneLineBlock             // label { ... } all on one line
	LineContinuation             // continuation of a multi-line value
	LineUnknown
)

// Line represents a single physical line of the source file.
type Line struct {
	Number  int
	Content []byte // without EOL
	Kind    LineKind
	Depth   int // nesting depth
}

// ItemKind distinguishes attributes from blocks.
type ItemKind int

const (
	ItemAttribute ItemKind = iota
	ItemBlock
)

// ItemModel represents a single attribute or block in the HCL body.
type ItemModel struct {
	Kind             ItemKind
	StartLine        int
	EndLine          int
	Name             string
	IsOneLine        bool // for blocks: { ... } on one line
	IsMultiLineValue bool // for attributes: value spans multiple lines
	ChildBody        *BodyModel
}

// Region groups an item with its leading comments and blank line count.
type Region struct {
	LeadingCommentLines []int // line numbers of leading comments
	Item                ItemModel
	BlanksBefore        int // blank lines before this region in original
}

// BodyModel represents the contents of an HCL body (top-level or nested).
type BodyModel struct {
	Depth      int
	StartLine  int
	EndLine    int
	IsTopLevel bool
	Regions    []Region
}

// FileModel is the top-level model for a parsed HCL file.
type FileModel struct {
	Path  string
	Src   []byte
	EOL   EOLKind
	Lines []Line
	Body  *BodyModel
}

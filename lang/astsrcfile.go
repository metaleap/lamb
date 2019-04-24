package atmolang

import (
	"github.com/go-leap/dev/lex"
	"github.com/go-leap/str"
	"github.com/metaleap/atmo"
)

type AstFiles []AstFile

type AstFile struct {
	TopLevel []AstFileTopLevelChunk
	errs     struct {
		loading error
	}
	LastLoad struct {
		Src                  []byte
		Time                 int64
		Size                 int64
		TokCountInitialGuess int
		NumLines             int
	}
	Options struct {
		ApplStyle ApplStyle
	}
	SrcFilePath string

	_toks udevlex.Tokens
	_errs []error
}

type AstFileTopLevelChunk struct {
	Src    []byte
	Offset struct {
		Line int
		Pos  int
	}
	id       [4]uint64
	_id      string
	srcDirty bool
	errs     struct {
		lexing  atmo.Errors
		parsing *atmo.Error
	}
	Ast AstTopLevel
}

func (me *AstFile) Errors() []error {
	if me._errs == nil {
		if me._errs = make([]error, 0); me.errs.loading != nil {
			me._errs = append(me._errs, me.errs.loading)
		}
		for i := range me.TopLevel {
			for e := range me.TopLevel[i].errs.lexing {
				me._errs = append(me._errs, &me.TopLevel[i].errs.lexing[e])
			}
			if e := me.TopLevel[i].errs.parsing; e != nil {
				me._errs = append(me._errs, e)
			}
		}
	}
	return me._errs
}

func (me *AstFile) String() (r string) {
	for i := range me.TopLevel {
		if def := me.TopLevel[i].Ast.Def; def != nil {
			r += "\n" + def.Tokens.String() + "\n"
		}
	}
	return
}

func (me *AstFile) CountTopLevelDefs() (total int, unexported int) {
	for i := range me.TopLevel {
		if ast := &me.TopLevel[i].Ast; ast.Def != nil {
			if total++; ast.DefIsUnexported {
				unexported++
			}
		}
	}
	return
}

func (me *AstFile) CountNetLinesOfCode() (sloc int) {
	var lastline int

	for i := range me.TopLevel {
		if def := me.TopLevel[i].Ast.Def; def != nil {
			for t := range def.Tokens {
				if tok := &def.Tokens[t]; tok.Meta.Line != lastline && tok.Kind() != udevlex.TOKEN_COMMENT {
					lastline, sloc = tok.Meta.Line, sloc+1
				}
			}
		}
	}
	return
}

func (me *AstFileTopLevelChunk) ID() string {
	if me._id == "" {
		me._id = ustr.Uint64s('-', me.id[:])
	}
	return me._id
}

func (me AstFiles) Len() int               { return len(me) }
func (me AstFiles) Swap(i int, j int)      { me[i], me[j] = me[j], me[i] }
func (me AstFiles) Less(i int, j int) bool { return me[i].SrcFilePath < me[j].SrcFilePath }

func (me AstFiles) Index(srcFilePath string) int {
	for i := range me {
		if me[i].SrcFilePath == srcFilePath {
			return i
		}
	}
	return -1
}

func (me *AstFiles) RemoveAt(idx int) {
	this := *me
	for i := idx; i < len(this)-1; i++ {
		this[i] = this[i+1]
	}
	this = this[:len(this)-1]
	*me = this
}
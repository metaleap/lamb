package atmocorefn

import (
	"github.com/metaleap/atmo"
	"github.com/metaleap/atmo/lang"
)

type IAstNode interface {
	Origin() atmolang.IAstNode
}

type IAstExpr interface {
	IAstNode
	IsAtomic() bool
}

type IAstExprAtomic interface {
	IAstExpr
	__implements_IAstExprAtomic()
}

type IAstIdent interface {
	IAstExprAtomic
}

type AstNodeBase struct {
}

type AstDefBase struct {
	AstNodeBase
	Orig *atmolang.AstDef

	Name IAstIdent
	Args []AstDefArg
	Body IAstExpr
}

type AstDef struct {
	AstDefBase
	Locals astDefs

	TopLevel *atmolang.AstFileTopLevelChunk
	Errs     atmo.Errors

	state struct {
		genNamePrefs  []string
		bodyWrapCases []struct{}
	}
}

func (me *AstDef) Origin() atmolang.IAstNode { return me.Orig }

type AstDefArg struct {
	AstIdentName

	Orig *atmolang.AstDefArg
}

type AstExprBase struct {
	AstNodeBase
}

func (*AstExprBase) IsAtomic() bool { return false }

type AstAtomBase struct {
	AstExprBase
}

func (*AstAtomBase) IsAtomic() bool               { return true }
func (*AstAtomBase) __implements_IAstExprAtomic() {}

type AstIdentBase struct {
	AstAtomBase
	Val string

	Orig *atmolang.AstIdent
}

func (me *AstIdentBase) Origin() atmolang.IAstNode { return me.Orig }

type AstIdentName struct {
	AstIdentBase
}

type AstIdentVar struct {
	AstIdentBase
}

type AstIdentTag struct {
	AstIdentBase
}

type AstIdentOp struct {
	AstIdentBase
}

type AstIdentEmptyParens struct {
	AstIdentBase
}

type AstIdentUnderscores struct {
	AstIdentBase
}

type AstLitBase struct {
	AstAtomBase
	Orig atmolang.IAstExprAtomic
}

func (me *AstLitBase) Origin() atmolang.IAstNode {
	return me.Orig
}

type AstLitRune struct {
	AstLitBase
	Val rune
}

type AstLitStr struct {
	AstLitBase
	Val string
}

type AstLitUint struct {
	AstLitBase
	Val uint64
}

type AstLitFloat struct {
	AstLitBase
	Val float64
}

func (me *AstIdentUnderscores) Num() int { return len(me.Val) }

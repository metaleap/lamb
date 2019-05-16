package atmolang

import (
	"strconv"

	"github.com/go-leap/dev/lex"
	"github.com/go-leap/str"
)

type IAstNode interface {
	print(*CtxPrint)
	Toks() udevlex.Tokens
}

type IAstComments interface {
	Comments() *astBaseComments
}

type IAstExpr interface {
	IAstNode
	IAstComments
	IsAtomic() bool
	Desugared(func() string) IAstExpr
}

type IAstExprAtomic interface {
	IAstExpr
	String() string
}

type AstBaseTokens struct {
	Tokens udevlex.Tokens
}

func (me *AstBaseTokens) Toks() udevlex.Tokens { return me.Tokens }

type astBaseComments = struct {
	Leading  AstComments
	Trailing AstComments
}

type AstBaseComments struct {
	comments astBaseComments
}

func (me *AstBaseComments) Comments() *astBaseComments {
	return &me.comments
}

type AstTopLevel struct {
	AstBaseTokens
	AstBaseComments
	Def struct {
		Orig         *AstDef
		IsUnexported bool
	}
}

type AstComments []AstComment

type AstComment struct {
	AstBaseTokens
	Val               string
	IsSelfTerminating bool
}

type AstDef struct {
	AstBaseTokens
	Name           AstIdent
	NameAffix      IAstExpr
	Args           []AstDefArg
	Meta           []IAstExpr
	Body           IAstExpr
	IsTopLevel     bool
	IsNakedAliasTo string
}

type AstDefArg struct {
	AstBaseTokens
	NameOrConstVal IAstExprAtomic
	Affix          IAstExpr
}

type AstBaseExpr struct {
	AstBaseTokens
	AstBaseComments
}

func (*AstBaseExpr) Desugared(func() string) IAstExpr { return nil }
func (*AstBaseExpr) IsAtomic() bool                   { return false }

type AstBaseExprAtom struct {
	AstBaseExpr
}

func (*AstBaseExprAtom) IsAtomic() bool { return true }

type AstBaseExprAtomLit struct {
	AstBaseExprAtom
}

type AstExprLitUint struct {
	AstBaseExprAtomLit
	Val uint64
}

func (me *AstExprLitUint) String() string { return strconv.FormatUint(me.Val, 10) }

type AstExprLitFloat struct {
	AstBaseExprAtomLit
	Val float64
}

func (me *AstExprLitFloat) String() string { return strconv.FormatFloat(me.Val, 'g', -1, 64) }

type AstExprLitRune struct {
	AstBaseExprAtomLit
	Val rune
}

func (me *AstExprLitRune) String() string { return strconv.QuoteRune(me.Val) }

type AstExprLitStr struct {
	AstBaseExprAtomLit
	Val string
}

func (me *AstExprLitStr) String() string { return strconv.Quote(me.Val) }

type AstIdent struct {
	AstBaseExprAtom
	Val     string
	IsOpish bool
	IsTag   bool
}

func (me *AstIdent) IsName(opishOk bool) bool {
	return ((!me.IsOpish) || opishOk) && (!me.IsTag) && me.Val[0] != '_'
}
func (me *AstIdent) String() string { return me.Val }

type AstExprAppl struct {
	AstBaseExpr
	Callee          IAstExpr
	Args            []IAstExpr
	HasPlaceholders bool
}

type AstExprLet struct {
	AstBaseExpr
	Defs []AstDef
	Body IAstExpr
}

type AstExprCases struct {
	AstBaseExpr
	Scrutinee    IAstExpr
	Alts         []AstCase
	defaultIndex int
}

type AstCase struct {
	AstBaseTokens
	Conds []IAstExpr
	Body  IAstExpr
}

func (me *AstComments) initFrom(accumComments []udevlex.Tokens) {
	this := make(AstComments, len(accumComments))
	for i := range accumComments {
		this[i].initFrom(accumComments[i], 0)
	}
	*me = this
}

func (me *AstComment) initFrom(tokens udevlex.Tokens, at int) {
	me.Tokens = tokens[at : at+1]
	me.Val, me.IsSelfTerminating = me.Tokens[0].Str, me.Tokens[0].IsCommentSelfTerminating()
}

func (me *AstExprCases) Default() *AstCase {
	if me.defaultIndex < 0 {
		return nil
	}
	return &me.Alts[me.defaultIndex]
}

func (me *AstExprCases) removeAltAt(idx int) {
	for i := idx; i < len(me.Alts)-1; i++ {
		me.Alts[i] = me.Alts[i+1]
	}
	me.Alts = me.Alts[:len(me.Alts)-1]
}

func (me *AstExprAppl) ClaspishByTokens() (claspish bool) {
	return len(me.Tokens) > 0 && (!me.Tokens.HasSpaces()) && !me.Tokens.HasKind(udevlex.TOKEN_COMMENT)
}

func (me *AstExprAppl) CalleeAndArgsOrdered(applStyle ApplStyle) (ret []IAstExpr) {
	ret = make([]IAstExpr, 1+len(me.Args))
	switch applStyle {
	case APPLSTYLE_VSO:
		for i := range me.Args {
			ret[i+1] = me.Args[i]
		}
		ret[0] = me.Callee
	case APPLSTYLE_SOV:
		for i := range me.Args {
			ret[i] = me.Args[i]
		}
		ret[len(ret)-1] = me.Callee
	case APPLSTYLE_SVO:
		for i := range me.Args {
			ret[i+1] = me.Args[i]
		}
		ret[0], ret[1] = me.Args[0], me.Callee
	}
	return
}

func (me *AstExprAppl) ToUnary() (unary *AstExprAppl) {
	if unary = me; len(me.Args) > 1 {
		appl := *me
		for len(appl.Args) > 1 {
			appl.Callee = &AstExprAppl{Callee: appl.Callee, Args: appl.Args[:1]}
			appl.Args = appl.Args[1:]
		}
		unary = &appl
	}
	return
}

func (me *AstDef) detectNakedAliasIfAny() {
	if len(me.Meta) == 0 && me.NameAffix == nil {
		if len(me.Args) == 0 {
			if ident, ok := me.Body.(*AstIdent); ok && ident != nil && ident.IsName(true) {
				me.IsNakedAliasTo = ident.Val
			}
		} else if appl, ok := me.Body.(*AstExprAppl); ok && appl != nil && len(appl.Args) == len(me.Args) {
			if ident, _ := appl.Callee.(*AstIdent); ident != nil && ident.IsName(true) {
				for i, arg := range appl.Args {
					daid, _ := me.Args[i].NameOrConstVal.(*AstIdent)
					aaid, _ := arg.(*AstIdent)
					if daid == nil || aaid == nil || daid.Val != aaid.Val {
						ok = false
						break
					}
				}
				if ok {
					me.IsNakedAliasTo = ident.Val
				}
			}
		}
	}
}

func (me *AstDef) Argless(prefix func() string) *AstDef {
	if len(me.Args) == 0 {
		return me
	}
	def, let := *me, &AstExprLet{Defs: make([]AstDef, 1)}
	let.Tokens, let.Defs[0].Tokens, let.Defs[0].Args, def.Args, let.Defs[0].Name =
		me.Tokens, me.Tokens, def.Args, nil, def.Name
	let.Defs[0].Name.Val, let.Defs[0].Body, let.Body, def.Body =
		prefix()+let.Defs[0].Name.Val, def.Body, &let.Defs[0].Name, let
	return &def
}

func (me *AstDef) makeUnary(origName string) {
	subname, let := ustr.Int(len(me.Args)-1)+origName, AstExprLet{Defs: []AstDef{{Body: me.Body, AstBaseTokens: me.AstBaseTokens}}}
	subdef := &let.Defs[0]
	let.AstBaseTokens, let.comments, me.Body, let.Body, me.Args, subdef.Args, subdef.Name.Val =
		me.AstBaseTokens, *me.Body.Comments(), &let, &subdef.Name, me.Args[:1], me.Args[1:], subname
	if len(subdef.Args) > 1 {
		subdef.makeUnary(origName)
	}
}

func (me *AstDef) ToUnary() *AstDef {
	if len(me.Args) <= 1 {
		return me
	}
	def := *me
	def.makeUnary(me.Name.Val)
	return &def
}

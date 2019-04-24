package atmocorefn

import (
	"github.com/go-leap/str"
	"github.com/metaleap/atmo"
	"github.com/metaleap/atmo/lang"
)

func (me *AstDef) initFrom(orig *atmolang.AstDef) {
	me.state.genNamePrefs = []string{orig.Name.Val}
	me.Errs.Add(me.AstDefBase.initFrom(me, orig))
}

func (me *AstDefBase) initFrom(ctx *AstDef, orig *atmolang.AstDef) (errs atmo.Errors) {
	me.Orig = orig
	errs.Add(me.initName(ctx))
	errs.Add(me.initBody(ctx))
	errs.Add(me.initArgs(ctx))
	errs.Add(me.initMetas(ctx))
	return
}

func (me *AstDefBase) initName(ctx *AstDef) (errs atmo.Errors) {
	tok := &me.Orig.Name.Tokens[0]
	if me.Name, errs = ctx.newAstIdentFrom(&me.Orig.Name); len(errs) == 0 {
		switch name := me.Name.(type) {
		case *AstIdentName:
			// all ok
		case *AstIdentOp:
			if name.Val == "" || ustr.In(name.Val, langReservedOps...) {
				errs.AddFrom(atmo.ErrCatNaming, tok, "reserved token not permissible as def name: `"+tok.Meta.Orig+"`")
			}
		case *AstIdentTag:
			errs.AddFrom(atmo.ErrCatNaming, tok, "invalid def name: `"+name.Val+"` is upper-case, this is reserved for tags")
		case *AstIdentVar:
			errs.AddFrom(atmo.ErrCatNaming, tok, "invalid def name: `"+tok.Meta.Orig+"` (begins with multiple underscores)")
		default:
			errs.AddFrom(atmo.ErrCatNaming, tok, "invalid def name: `"+tok.Meta.Orig+"`")
		}
	}
	return
}

func (me *AstDefBase) initBody(ctx *AstDef) (errs atmo.Errors) {
	me.Body, errs = ctx.newAstExprFrom(me.Orig.Body)
	return
}

func (me *AstDefBase) initArgs(ctx *AstDef) (errs atmo.Errors) {
	me.Args = make([]AstDefArg, len(me.Orig.Args))
	for i := range me.Orig.Args {
		errs.Add(me.Args[i].initFrom(ctx, &me.Orig.Args[i], i))
	}
	return
}

func (me *AstDefBase) initMetas(ctx *AstDef) (errs atmo.Errors) {
	// TODO wrap
	return
}

func (me *AstDef) newAstIdentFrom(orig *atmolang.AstIdent) (ident IAstIdent, errs atmo.Errors) {
	if orig.IsTag || ustr.BeginsUpper(orig.Val) {
		var tag AstIdentTag
		ident, errs = &tag, tag.initFrom(me, orig)

	} else if orig.IsOpish {
		if orig.Val == "()" {
			var empar AstIdentEmptyParens
			ident = &empar
		} else {
			var op AstIdentOp
			ident, errs = &op, op.initFrom(me, orig)
		}

	} else if orig.Val[0] != '_' {
		var name AstIdentName
		ident, errs = &name, name.initFrom(me, orig)

	} else if ustr.IsRepeat(orig.Val) {
		var unsco AstIdentUnderscores
		ident, errs = &unsco, unsco.initFrom(me, orig)

	} else if orig.Val[1] != '_' {
		var idvar AstIdentVar
		ident, errs = &idvar, idvar.initFrom(me, orig)

	} else {
		errs.AddFrom(atmo.ErrCatNaming, &orig.Tokens[0], "invalid identifier: begins with multiple underscores")
	}
	return
}

func (me *AstDef) newAstExprFrom(orig atmolang.IAstExpr) (expr IAstExpr, errs atmo.Errors) {
	switch o := orig.(type) {
	case *atmolang.AstExprLitFloat:
		var lit AstLitFloat
		lit.initFrom(me, o)
		expr = &lit
	case *atmolang.AstExprLitUint:
		var lit AstLitUint
		lit.initFrom(me, o)
		expr = &lit
	case *atmolang.AstExprLitRune:
		var lit AstLitRune
		lit.initFrom(me, o)
		expr = &lit
	case *atmolang.AstExprLitStr:
		var lit AstLitStr
		lit.initFrom(me, o)
		expr = &lit
	case *atmolang.AstIdent:
		expr, errs = me.newAstIdentFrom(o)
	case *atmolang.AstExprLet:
		expr, errs = me.newAstExprFrom(o.Body)
		for i := range o.Defs {
			var def AstDefBase
			if !errs.Add(def.initFrom(me, &o.Defs[i])) {
				me.Locals = append(me.Locals, def)
			}
		}
	case *atmolang.AstExprAppl:
	case *atmolang.AstExprCase:
	default:
		panic(o)
	}
	return
}

func (me *AstIdentBase) initFrom(ctx *AstDef, from *atmolang.AstIdent) (errs atmo.Errors) {
	me.Val = from.Val
	return
}

func (me *AstDefArg) initFrom(ctx *AstDef, orig *atmolang.AstDefArg, argIdx int) (errs atmo.Errors) {
	me.Orig = orig

	var constexpr IAstExprAtomic
	switch v := orig.NameOrConstVal.(type) {
	case *atmolang.AstIdent:
		if constexpr, errs = ctx.newAstIdentFrom(v); constexpr != nil {
			if cx, ok := constexpr.(*AstIdentName); ok {
				constexpr, me.AstIdentName = nil, *cx
			}
		}
	case *atmolang.AstExprLitFloat, *atmolang.AstExprLitUint, *atmolang.AstExprLitRune, *atmolang.AstExprLitStr:
		constexpr, errs = ctx.newAstExprFrom(v)
	default:
		panic(v)
	}
	if constexpr != nil {
		me.AstIdentName.Val = "~arg~" + ustr.Int(argIdx)
		// TODO wrap
	}

	if orig.Affix != nil {
		// TODO wrap
	}
	return
}

func (me *AstLitBase) initFrom(ctx *AstDef, orig atmolang.IAstExprAtomic) {
	me.Orig = orig
}

func (me *AstLitFloat) initFrom(ctx *AstDef, orig atmolang.IAstExprAtomic) {
	me.AstLitBase.initFrom(ctx, orig)
	me.Val = orig.BaseTokens().Tokens[0].Float
}

func (me *AstLitUint) initFrom(ctx *AstDef, orig atmolang.IAstExprAtomic) {
	me.AstLitBase.initFrom(ctx, orig)
	me.Val = orig.BaseTokens().Tokens[0].Uint
}

func (me *AstLitRune) initFrom(ctx *AstDef, orig atmolang.IAstExprAtomic) {
	me.AstLitBase.initFrom(ctx, orig)
	me.Val = orig.BaseTokens().Tokens[0].Rune()
}

func (me *AstLitStr) initFrom(ctx *AstDef, orig atmolang.IAstExprAtomic) {
	me.AstLitBase.initFrom(ctx, orig)
	me.Val = orig.BaseTokens().Tokens[0].Str
}

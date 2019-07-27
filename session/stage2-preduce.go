package atmosess

import (
	"strconv"

	"github.com/go-leap/str"
	"github.com/metaleap/atmo"
	"github.com/metaleap/atmo/il"
)

type IPreduced interface {
	self() *preducedBase
	SummaryCompact() string
}

type preducedBase struct {
	origNode atmoil.INode
}

func (me *preducedBase) self() *preducedBase { return me }

type PConstValAtomic struct {
	preducedBase
	LitVal interface{}
}

func (me *PConstValAtomic) SummaryCompact() string {
	switch lv := me.LitVal.(type) {
	case float64:
		return strconv.FormatFloat(lv, 'f', -1, 64)
	case uint64:
		return strconv.FormatUint(lv, 10)
	case string: // tag
		return lv
	default:
		panic(lv)
	}
}

type PConstValCompound struct {
	preducedBase

	// later will be more principled as compound will encompass lists-of-any, tuples, relations / maps / records etc.
	TmpVal string
}

func (me *PConstValCompound) SummaryCompact() string {
	return strconv.Quote(me.TmpVal)
}

type PFailure struct {
	preducedBase

	ErrMsg string
}

func (me *PFailure) SummaryCompact() string { return me.ErrMsg }

type PCallable struct {
	preducedBase
}

type ctxPreduce struct {
	owner *Ctx
}

func (me *Ctx) PreduceExpr(kit *Kit, maybeTopDefId string, expr atmoil.IExpr) (IPreduced, atmo.Errors) {
	var maybetld *atmoil.IrDefTop
	if maybeTopDefId != "" {
		maybetld = kit.lookups.tlDefsByID[maybeTopDefId]
	}
	return me.state.preduce.preduceIlNode(kit, maybetld, expr)
}

func (me *ctxPreduce) preduceIlNode(kit *Kit, maybeTld *atmoil.IrDefTop, node atmoil.INode) (ret IPreduced, errs atmo.Errors) {
	switch this := node.(type) {
	case *atmoil.IrLitFloat:
		ret = &PConstValAtomic{LitVal: this.Val}
	case *atmoil.IrLitUint:
		ret = &PConstValAtomic{LitVal: this.Val}
	case *atmoil.IrIdentTag:
		ret = &PConstValAtomic{LitVal: this.Val}
	case *atmoil.IrLitStr:
		ret = &PConstValCompound{TmpVal: this.Val}
	case *atmoil.IrIdentName:
		cands := kit.lookups.namesInScopeAll[this.Val]
		if len(cands) == 0 {
			ret = &PFailure{ErrMsg: "not defined or not in scope: `" + this.Val + "`"}
		} else if len(cands) == 1 {
			refkit := kit
			if extref, ok := cands[0].(IrDefRef); ok {
				me.owner.KitsEnsureLoaded(false, extref.KitImpPath)
				if refkit = me.owner.KitByImpPath(extref.KitImpPath); refkit == nil {
					ret = &PFailure{ErrMsg: "no such kit known: `" + extref.KitImpPath + "`"}
				}
			} else {
				ret = &PFailure{ErrMsg: "ambiguous: `" + this.Val + "` (" + ustr.Int(len(cands)) + " such names in scope)"}
			}
			if refkit != nil {
				ret, errs = me.preduceIlNode(kit, maybeTld, cands[0])
			}
		}
	case *atmoil.IrDefTop:
		ret, errs = me.preduceIlNode(kit, this, &this.IrDef)
	case *atmoil.IrDef:
		ret, errs = me.preduceIlNode(kit, maybeTld, this.Body)
	default:
		panic(this)
	}
	ret.self().origNode = node
	return
}

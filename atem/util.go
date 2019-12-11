package atem

func callArgs(argsInIntuitiveOrder ...Expr) (argsInReverseOrder []Expr) {
	argsInReverseOrder = make([]Expr, len(argsInIntuitiveOrder))
	for i, argexpr := range argsInIntuitiveOrder {
		argsInReverseOrder[len(argsInReverseOrder)-(i+1)] = argexpr
	}
	return
}

// Eq is the fallback for `OpEq` calls with 2 operands that aren't both `ExprNumInt`s.
func (me Prog) Eq(expr Expr, cmp Expr) bool {
	if expr == cmp {
		return true
	} else {
		switch it := expr.(type) {
		case ExprNumInt:
			that, ok := cmp.(ExprNumInt)
			return ok && it == that
		case ExprArgRef:
			that, ok := cmp.(ExprArgRef)
			return ok && (it == that || (it < 0 && that >= 0 && that == (-it)-1) || (it >= 0 && that < 0 && it == (-that)-1))
		case ExprFuncRef:
			that, ok := cmp.(ExprFuncRef)
			return ok && it == that
		case *ExprCall:
			if that, ok := cmp.(*ExprCall); ok {
				ok = me.Eq(it.Callee, that.Callee)
				if ok = ok && len(it.Args) == len(that.Args); ok {
					for i := 0; ok && i < len(it.Args) && i < len(that.Args); i++ {
						ok = me.Eq(it.Args[i], that.Args[i])
					}
				}
				return ok
			}
		}
	}
	return false
}

// ListOfExprs dissects the given `expr` into an `[]Expr` slice only if it is
// a closure resulting from `StdFuncCons` / `StdFuncNil` usage during `Eval`.
// The individual element `Expr`s are not themselves scrutinized however.
// The `ret` is `return`ed as `nil` if `expr` isn't a product of `StdFuncCons`
// / `StdFuncNil` usage; yet a non-`nil`, zero-`len` `ret` will result from a
// mere `StdFuncNil` construction, aka. "empty linked-list value" `Expr`.
func (me Prog) ListOfExprs(expr Expr) (ret []Expr) {
	ret = make([]Expr, 0, 1024)
	for ok, next := true, me.eval(expr, nil); ok; {
		ok = false
		if fnref, _ := next.(ExprFuncRef); fnref == StdFuncNil {
			break
		} else if call, okc := next.(*ExprCall); okc && len(call.Args) == 2 {
			if fnref, _ = call.Callee.(ExprFuncRef); fnref == StdFuncCons {
				curEvalDepth++
				for i := len(call.Args) - 1; i > 0; i-- {
					ret = append(ret, me.eval(call.Args[i], nil))
				}
				ok, next = true, me.eval(call.Args[0], nil)
				curEvalDepth--
			}
		}
		if !ok {
			ret = nil
		}
	}
	return
}

// ListToBytes examines the given `[]Expr`, as normally obtained via
// `Prog.ListOfExprs` and accumulates a `[]byte` slice as long as all elements
// in said list are `ExprNumInt` values in the range 0 - 255. If the input is
// `nil`, so will be `retNumListAsBytes`. If the input has a `len` of zero,
// so will `retNumListAsBytes`. If any of the input `Expr`s isn't an in-range
// `ExprNumInt`, then too will `retNumListAsBytes` be `nil`.
func ListToBytes(maybeNumList []Expr) (retNumListAsBytes []byte) {
	if maybeNumList != nil {
		retNumListAsBytes = make([]byte, 0, len(maybeNumList))
		for _, expr := range maybeNumList {
			if num, ok := expr.(ExprNumInt); ok && num > -1 && num < 256 {
				retNumListAsBytes = append(retNumListAsBytes, byte(num))
			} else {
				retNumListAsBytes = nil
				break
			}
		}
	}
	return
}

// ListOfExprsToString is a wrapper around the combined usage of `Prog.ListOfExprs`
// and `ListToBytes` to extract the List-closure-encoded `string` of an `Eval`
// result, if it is one. Otherwise, `expr.JsonSrc()` is returned for convenience.
func (me Prog) ListOfExprsToString(expr Expr) string {
	if maybenumlist := me.ListOfExprs(expr); maybenumlist != nil {
		if bytes := ListToBytes(maybenumlist); bytes != nil {
			return string(bytes)
		}
	}
	return expr.JsonSrc()
}

// ListFrom converts the specified byte string to a linked-list representing a text string during `Eval` (via `ExprCall`s of `StdFuncCons` and `StdFuncNil`).
func ListFrom(str []byte) (ret Expr) {
	ret = StdFuncNil
	for i := len(str) - 1; i > -1; i-- {
		ret = &ExprCall{isClosure: true, allArgsDone: true, hasArgRefs: false, Callee: StdFuncCons, Args: []Expr{ret, ExprNumInt(str[i])}}
		// ret = &ExprCall{Callee: &ExprCall{Callee: StdFuncCons, Args: []Expr{ExprNumInt(str[i])}}, Args: []Expr{ret}}
	}
	return
}

// ListsFrom creates from `strs` linked-lists via `ListFrom`, and returns a linked-list of those.
func ListsFrom(strs []string) (ret Expr) {
	ret = StdFuncNil
	for i := len(strs) - 1; i > -1; i-- {
		ret = &ExprCall{isClosure: true, allArgsDone: true, hasArgRefs: false, Callee: StdFuncCons, Args: []Expr{ret, ListFrom([]byte(strs[i]))}}
		// ret = &ExprCall{Callee: &ExprCall{Callee: StdFuncCons, Args: []Expr{ListFrom([]byte(strs[i]))}}, Args: []Expr{ret}}
	}
	return
}

// This is neither used by the evaluator nor by the parser but occasionally
// handy temporarily for investigating, profiling or optimizing specifics.
func walk(expr Expr, visitor func(Expr) Expr) Expr {
	if ret := visitor(expr); ret != nil {
		expr = ret
		if call, ok := expr.(*ExprCall); ok {
			call.Callee = walk(call.Callee, visitor)
			for i, argval := range call.Args {
				call.Args[i] = walk(argval, visitor)
			}
			expr = call
		}
	}
	return expr
}
